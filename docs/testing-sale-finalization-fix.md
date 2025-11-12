# Testing Sale Finalization Race Condition Fix

## Quick Manual Test

### Test 1: Verify Locking Works

```sql
-- Terminal 1: Start a transaction and lock cart items
BEGIN;
SELECT SUM(total_price) - SUM(discount_amount) AS sum 
FROM (
    SELECT total_price, discount_amount 
    FROM cart_items 
    WHERE sale_id = 'YOUR_SALE_ID' 
    FOR UPDATE
) AS locked_items;
-- Don't commit yet!

-- Terminal 2: Try to add a cart item (should wait/block)
-- This should block until Terminal 1 commits
INSERT INTO cart_items (sale_id, store_product_id, unit_quantity, unit_price, total_price)
VALUES ('YOUR_SALE_ID', 'SOME_PRODUCT_ID', 10, 100.00, 100.00);

-- Terminal 1: Now commit
COMMIT;

-- Terminal 2: Should now complete
```

### Test 2: Verify Aggregate Queries Work

```sql
-- Test the subquery approach works correctly
SELECT SUM(total_price) - SUM(discount_amount) AS sum 
FROM (
    SELECT total_price, discount_amount 
    FROM cart_items 
    WHERE sale_id = 'YOUR_SALE_ID' 
    FOR UPDATE
) AS locked_items;

-- Should return the correct sum without errors
```

### Test 3: Verify Complex Aggregate Query

```sql
-- Test GetCartItemsTotalAmount query
SELECT 
    SUM(locked_ci.total_price) AS sum,
    SUM(locked_ci.unit_quantity/p.unit_per_pack) AS item_count,
    SUM(locked_ci.discount_amount) AS discount_amount,
    ROUND(SUM((sp.vat_price / p.unit_per_pack) * locked_ci.unit_quantity), 2) AS vat_sum,
    SUM(locked_ci.total_price) - SUM(locked_ci.discount_amount) as total_amount
FROM (
    SELECT * FROM cart_items WHERE sale_id = 'YOUR_SALE_ID' FOR UPDATE
) locked_ci
JOIN store_products sp ON locked_ci.store_product_id = sp.id
JOIN products p ON sp.product_id = p.id;

-- Should return correct aggregates without errors
```

## Integration Test Scenarios

### Scenario 1: Concurrent Cart Item Addition During Finalization

```go
func TestConcurrentCartItemAdditionDuringFinalization(t *testing.T) {
    // Setup: Create a sale with cart items
    saleID := createTestSale()
    addCartItem(saleID, 100.00)
    
    // Start finalization in goroutine
    finalizeDone := make(chan error)
    go func() {
        finalizeDone <- finalizeSale(saleID, 100.00)
    }()
    
    // Try to add cart item concurrently
    time.Sleep(50 * time.Millisecond) // Let finalization start
    addErr := addCartItem(saleID, 50.00)
    
    // Wait for finalization
    finalizeErr := <-finalizeDone
    
    // Assertions
    assert.NoError(t, finalizeErr, "Finalization should succeed")
    assert.Error(t, addErr, "Adding cart item should fail")
    assert.Equal(t, domain.SaleIsClosedError, addErr, "Should get SaleIsClosedError")
    
    // Verify sale total matches cart items
    sale := getSale(saleID)
    cartSum := getCartItemsSum(saleID)
    assert.Equal(t, 100.00, sale.TotalAmount)
    assert.Equal(t, sale.TotalAmount, cartSum, "Sale total should match cart items sum")
}
```

### Scenario 2: Concurrent Cart Item Update During Finalization

```go
func TestConcurrentCartItemUpdateDuringFinalization(t *testing.T) {
    // Setup: Create a sale with cart items
    saleID := createTestSale()
    cartItemID := addCartItem(saleID, 100.00)
    
    // Start finalization in goroutine
    finalizeDone := make(chan error)
    go func() {
        finalizeDone <- finalizeSale(saleID, 100.00)
    }()
    
    // Try to update cart item concurrently
    time.Sleep(50 * time.Millisecond)
    updateErr := updateCartItemQuantity(cartItemID, 2) // Double quantity
    
    // Wait for finalization
    finalizeErr := <-finalizeDone
    
    // Assertions
    assert.NoError(t, finalizeErr, "Finalization should succeed")
    assert.Error(t, updateErr, "Updating cart item should fail or wait")
    
    // Verify sale total is correct
    sale := getSale(saleID)
    assert.Equal(t, 100.00, sale.TotalAmount)
}
```

### Scenario 3: Concurrent Cart Item Deletion During Finalization

```go
func TestConcurrentCartItemDeletionDuringFinalization(t *testing.T) {
    // Setup: Create a sale with multiple cart items
    saleID := createTestSale()
    cartItem1 := addCartItem(saleID, 100.00)
    cartItem2 := addCartItem(saleID, 50.00)
    
    // Start finalization in goroutine
    finalizeDone := make(chan error)
    go func() {
        finalizeDone <- finalizeSale(saleID, 150.00)
    }()
    
    // Try to delete cart item concurrently
    time.Sleep(50 * time.Millisecond)
    deleteErr := deleteCartItem(cartItem2)
    
    // Wait for finalization
    finalizeErr := <-finalizeDone
    
    // Assertions
    assert.NoError(t, finalizeErr, "Finalization should succeed")
    assert.Error(t, deleteErr, "Deleting cart item should fail")
    
    // Verify sale total includes both items
    sale := getSale(saleID)
    assert.Equal(t, 150.00, sale.TotalAmount)
}
```

### Scenario 4: Multiple Concurrent Finalizations (Should Not Happen)

```go
func TestMultipleConcurrentFinalizations(t *testing.T) {
    // Setup: Create a sale with cart items
    saleID := createTestSale()
    addCartItem(saleID, 100.00)
    
    // Try to finalize twice concurrently
    finalize1Done := make(chan error)
    finalize2Done := make(chan error)
    
    go func() {
        finalize1Done <- finalizeSale(saleID, 100.00)
    }()
    
    go func() {
        time.Sleep(10 * time.Millisecond) // Slight delay
        finalize2Done <- finalizeSale(saleID, 100.00)
    }()
    
    // Wait for both
    err1 := <-finalize1Done
    err2 := <-finalize2Done
    
    // Assertions: One should succeed, one should fail
    successCount := 0
    if err1 == nil {
        successCount++
    }
    if err2 == nil {
        successCount++
    }
    
    assert.Equal(t, 1, successCount, "Only one finalization should succeed")
    
    // Verify sale is finalized correctly
    sale := getSale(saleID)
    assert.Equal(t, 100.00, sale.TotalAmount)
    assert.Equal(t, constants.SaleStageFinished, sale.Stage)
}
```

## Load Test

### Simulate High Concurrency

```go
func TestHighConcurrencyFinalization(t *testing.T) {
    numSales := 100
    concurrentOps := 10 // Operations per sale
    
    for i := 0; i < numSales; i++ {
        saleID := createTestSale()
        addCartItem(saleID, 100.00)
        
        var wg sync.WaitGroup
        wg.Add(concurrentOps)
        
        // Start finalization
        go func() {
            defer wg.Done()
            finalizeSale(saleID, 100.00)
        }()
        
        // Try concurrent operations
        for j := 0; j < concurrentOps-1; j++ {
            go func() {
                defer wg.Done()
                // Random operation
                switch rand.Intn(3) {
                case 0:
                    addCartItem(saleID, 50.00)
                case 1:
                    updateCartItemQuantity(getFirstCartItem(saleID), 2)
                case 2:
                    deleteCartItem(getFirstCartItem(saleID))
                }
            }()
        }
        
        wg.Wait()
        
        // Verify consistency
        sale := getSale(saleID)
        cartSum := getCartItemsSum(saleID)
        assert.Equal(t, sale.TotalAmount, cartSum, 
            "Sale %s: total_amount should match cart items sum", saleID)
    }
}
```

## Expected Results

### Before Fix:
- ❌ Race conditions occur
- ❌ `sale.total_amount` ≠ `SUM(cart_items.total_price)`
- ❌ Data inconsistency

### After Fix:
- ✅ No race conditions
- ✅ `sale.total_amount` = `SUM(cart_items.total_price)` always
- ✅ Cart item operations fail with `SaleIsClosedError` after finalization starts
- ✅ Only one finalization succeeds per sale
- ✅ Data consistency maintained

## Performance Benchmarks

Run these to ensure locking doesn't significantly impact performance:

```go
func BenchmarkFinalizeSaleWithLocking(b *testing.B) {
    for i := 0; i < b.N; i++ {
        saleID := createTestSale()
        addCartItem(saleID, 100.00)
        finalizeSale(saleID, 100.00)
    }
}

func BenchmarkConcurrentSaleOperations(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            saleID := createTestSale()
            addCartItem(saleID, 100.00)
            finalizeSale(saleID, 100.00)
        }
    })
}
```

## Monitoring Queries

### Check for Locked Transactions

```sql
-- See currently locked cart_items
SELECT 
    l.pid,
    l.mode,
    l.granted,
    a.query,
    a.state,
    a.wait_event_type,
    a.wait_event
FROM pg_locks l
JOIN pg_stat_activity a ON l.pid = a.pid
WHERE l.relation = 'cart_items'::regclass
ORDER BY l.granted, a.query_start;
```

### Check for Deadlocks

```sql
-- Check PostgreSQL logs for deadlocks
SELECT * FROM pg_stat_database_conflicts WHERE datname = 'your_database';
```

### Monitor Transaction Duration

```sql
-- Find long-running transactions
SELECT 
    pid,
    now() - xact_start AS duration,
    state,
    query
FROM pg_stat_activity
WHERE state != 'idle'
    AND xact_start IS NOT NULL
ORDER BY duration DESC;
```
