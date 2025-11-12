# Sale Finalization Race Condition Fix

## Problem Description

After adding locking logic to the sale finalization process, there was still a race condition where:
- New cart items could be added during sale finalization
- Existing cart items could be modified or deleted during sale finalization
- This caused the sale's `total_amount` to not match the sum of cart_items' `total_price`

## Root Cause Analysis

The issue was that while we locked the **sale** record using `GetSaleByIdWithLocking()`, we did **NOT lock the cart_items** associated with that sale.

### The Race Condition Flow:

```
Time | Thread A (Finalize Sale)              | Thread B (Add/Update Cart Item)
-----|----------------------------------------|----------------------------------
T1   | Lock sale record                       |
T2   | Read cart_items sum = 1000             |
T3   |                                        | Add new cart item (+100)
T4   |                                        | Cart items sum now = 1100
T5   | Save sale with total_amount = 1000     |
T6   | Commit transaction                     |
T7   |                                        | Commit transaction
-----|----------------------------------------|----------------------------------
Result: Sale total_amount (1000) ≠ Cart items sum (1100) ❌
```

### Why This Happened:

1. **In `FinalizeSale()`**: Only the sale row was locked with `FOR UPDATE`
2. **In `matchingPaymentTypeSum()`**: Called `cartItemsSumBySaleId()` which read cart_items **without locking**
3. **In `validateSaleProductQuantity()`**: Read cart_items **without locking**
4. **In `ApplySaleInventoryUpdate()`**: Read cart_items **without locking**
5. **Meanwhile**: Other transactions could still modify cart_items because they were not locked

## Solution

Add `FOR UPDATE` (row-level locking) to **all cart_items queries** within the sale finalization transaction.

### PostgreSQL Limitations

#### Limitation 1: FOR UPDATE with Aggregate Functions

PostgreSQL doesn't allow `FOR UPDATE` directly with aggregate functions like `SUM()`, `COUNT()`, etc. This query will fail:

```sql
-- ❌ This will fail with: FOR UPDATE is not allowed with aggregate functions
SELECT SUM(total_price) FROM cart_items WHERE sale_id = ? FOR UPDATE
```

**Workaround**: Use a subquery to lock the rows first, then aggregate in the outer query:

```sql
-- ✅ This works: lock rows in subquery, aggregate in outer query
SELECT SUM(total_price) 
FROM (
    SELECT total_price FROM cart_items WHERE sale_id = ? FOR UPDATE
) AS locked_items
```

#### Limitation 2: FOR UPDATE with LEFT JOIN (Outer Joins)

PostgreSQL doesn't allow `FOR UPDATE` on the nullable side of an outer join. This query will fail:

```sql
-- ❌ This will fail with: FOR UPDATE cannot be applied to the nullable side of an outer join
SELECT ci.*, pb.bonus_amount
FROM cart_items ci
LEFT JOIN product_bonuses pb ON ci.product_id = pb.product_id
WHERE ci.sale_id = ?
FOR UPDATE
```

**Workaround**: Use `FOR UPDATE OF` to specify which table(s) to lock:

```sql
-- ✅ This works: lock only cart_items, not the LEFT JOINed table
SELECT ci.*, pb.bonus_amount
FROM cart_items ci
LEFT JOIN product_bonuses pb ON ci.product_id = pb.product_id
WHERE ci.sale_id = ?
FOR UPDATE OF ci
```

This approach:
1. Locks only the `cart_items` table rows
2. Does not attempt to lock the `product_bonuses` table (which may have NULL rows)
3. Other transactions are blocked from modifying cart_items until commit

### Changes Made:

#### 1. Lock cart items when calculating sum (`sale.service.go`)
```go
// Before:
SELECT SUM(total_price) - SUM(discount_amount) AS sum 
FROM cart_items WHERE sale_id = ?

// After (using subquery because PostgreSQL doesn't allow FOR UPDATE with aggregates):
SELECT SUM(total_price) - SUM(discount_amount) AS sum 
FROM (
    SELECT total_price, discount_amount 
    FROM cart_items 
    WHERE sale_id = ? 
    FOR UPDATE
) AS locked_items
```

**Note**: PostgreSQL doesn't allow `FOR UPDATE` directly with aggregate functions, so we use a subquery to lock the rows first, then aggregate in the outer query.

#### 2. Lock cart items when validating quantities (`sale.service.go`)
```go
// Using FOR UPDATE OF to lock only cart_items (not LEFT JOINed tables)
err := tx.WithContext(ctx).Raw(`
    SELECT ci.*, sp.unit_quantity, p.unit_per_pack, pb.bonus_amount
    FROM cart_items ci
    JOIN store_products sp ON sp.id = ci.store_product_id
    JOIN products p ON sp.product_id = p.id
    LEFT JOIN product_bonuses pb ON p.id = pb.product_id
    WHERE ci.sale_id = ?
    FOR UPDATE OF ci
`, sale.Id).Scan(&cartItemsWithProducts).Error
```

#### 3. Lock cart items when applying inventory updates (`sale.service.go`)
```go
// Using FOR UPDATE OF to lock only cart_items (not LEFT JOINed tables)
err := tx.WithContext(ctx).Raw(`
    SELECT ci.*, sp.unit_quantity, p.unit_per_pack, pb.bonus_amount
    FROM cart_items ci
    JOIN store_products sp ON sp.id = ci.store_product_id
    JOIN products p ON sp.product_id = p.id
    LEFT JOIN product_bonuses pb ON p.id = pb.product_id
    WHERE ci.sale_id = ?
    FOR UPDATE OF ci
`, sale.Id).Scan(&cartItemsWithProducts).Error
```

#### 4. Lock cart items in `GetCartItemsTotalAmount()` (`cart_item.service.go`)
```go
// Using subquery to lock rows before aggregation
err := tx.WithContext(ctx).Raw(`
    SELECT 
        SUM(locked_ci.total_price) AS sum,
        SUM(locked_ci.unit_quantity/p.unit_per_pack) AS item_count,
        -- ... other aggregates ...
    FROM (
        SELECT * FROM cart_items WHERE sale_id = ? FOR UPDATE
    ) locked_ci
    JOIN store_products sp ON locked_ci.store_product_id = sp.id
    JOIN products p ON sp.product_id = p.id
`, saleId).Scan(&res).Error
```

#### 5. Lock cart items in `getCartItemWithProducts()` (`cart_item.service.go`)
```go
// Using FOR UPDATE OF to lock only cart_items
err := tx.WithContext(ctx).Raw(`
    SELECT ci.*, p.name, p.barcode
    FROM cart_items ci
    JOIN store_products sp ON ci.store_product_id = sp.id
    JOIN products p ON sp.product_id = p.id
    WHERE ci.sale_id = ?
    FOR UPDATE OF ci
`, saleId).Scan(&cartItems).Error
```

## How This Fixes The Problem

### New Flow with Proper Locking:

```
Time | Thread A (Finalize Sale)              | Thread B (Add/Update Cart Item)
-----|----------------------------------------|----------------------------------
T1   | Lock sale record                       |
T2   | Lock all cart_items (FOR UPDATE)       |
T3   |                                        | Try to add cart item
T4   |                                        | BLOCKED - waiting for lock
T5   | Calculate sum = 1000                   | (still waiting...)
T6   | Save sale with total_amount = 1000     | (still waiting...)
T7   | Commit transaction (releases locks)    | (still waiting...)
T8   |                                        | Lock acquired, add cart item
T9   |                                        | But sale is already finalized!
T10  |                                        | Check fails: SaleIsClosedError ✓
-----|----------------------------------------|----------------------------------
Result: Sale total_amount (1000) = Cart items sum (1000) ✅
```

### Key Benefits:

1. **Atomicity**: All cart_items are locked during finalization
2. **Consistency**: No cart items can be added/modified/deleted during finalization
3. **Isolation**: Other transactions must wait until finalization completes
4. **Durability**: Once committed, the sale and cart_items are in sync

## Testing Recommendations

### Unit Tests:
- Test concurrent cart item additions during finalization
- Test concurrent cart item updates during finalization
- Test concurrent cart item deletions during finalization

### Integration Tests:
- Simulate multiple users trying to modify the same sale simultaneously
- Verify that only one finalization succeeds
- Verify that cart item operations fail with `SaleIsClosedError` after finalization

### Load Tests:
- Run concurrent finalization requests for different sales
- Verify no deadlocks occur
- Verify performance is acceptable with locking

## Performance Considerations

### Potential Impact:
- **Lock contention**: If multiple users try to modify the same sale, they will wait
- **Transaction duration**: Longer transactions hold locks longer

### Mitigation:
- Keep finalization transaction as short as possible
- Ensure proper indexes on `cart_items.sale_id`
- Monitor for deadlocks in production

### Database Indexes:
```sql
-- Ensure this index exists for efficient locking
CREATE INDEX IF NOT EXISTS idx_cart_items_sale_id ON cart_items(sale_id);
```

## Related Files Modified

1. `internal/services/sale.service.go`
   - `cartItemsSumBySaleId()` - Added FOR UPDATE
   - `validateSaleProductQuantity()` - Added Clauses(clause.Locking)
   - `ApplySaleInventoryUpdate()` - Added Clauses(clause.Locking)

2. `internal/services/cart_item.service.go`
   - `GetCartItemsTotalAmount()` - Added Clauses(clause.Locking)
   - `getCartItemWithProducts()` - Added Clauses(clause.Locking)

## Deployment Notes

- This is a **critical bug fix** for data consistency
- No database migrations required
- No API changes required
- Backward compatible with existing code
- Should be deployed ASAP to prevent data inconsistencies

## Monitoring

After deployment, monitor for:
- Reduced instances of `total_amount` mismatches
- No increase in database deadlocks
- Acceptable transaction response times
- No increase in `SaleIsClosedError` errors (some increase is expected and correct)
