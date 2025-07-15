# Technical Design: Final Sale Process

## Overview
The Final Sale Process is a critical business operation that handles the completion of sales transactions, including payment processing, inventory updates, and transaction finalization. This document outlines the technical architecture, flow, and implementation details.

## System Architecture

### Components
- **Sale Handler**: HTTP controller managing the sale finalization endpoint
- **Payment Services**: Integration with multiple payment providers (Click, Payme, Uzum, Alif)
- **Transaction Manager**: Database transaction coordination
- **Inventory Service**: Stock management and marking validation
- **Concurrency Control**: Mutex-based locking for parallel request handling

## API Specification

### Endpoint
```
POST /sale/final
```

### Request Structure
```json
{
  "sale_id": "string",
  "store_id": "string",
  "total_amount": "number",
  "cash_box_operation_id": "string",
  "payment_types": [
    {
      "type": "cash|card|app",
      "app_type": "click|payme|uzum|alif",
      "amount": "number"
    }
  ],
  "marking_data": "object"
}
```

## Process Flow

### 1. Request Validation
- JSON binding validation
- Payment types presence check
- Amount validation across multiple dimensions

### 2. Concurrency Control
```go
mu := h.getOrderLock(body.SaleID)
mu.Lock()
defer mu.Unlock()
```
- Prevents parallel processing of the same sale
- Uses sale ID as lock key

### 3. Database Transaction Management
- Single transaction scope for entire operation
- Automatic rollback on any failure
- Commit only after all operations succeed

### 4. Sale Status Validation
- Retrieve sale record from database
- Verify sale exists and is not already completed
- Prevent duplicate processing

### 5. Inventory Management
- Add marking count to cart items
- Validate stock availability
- Update inventory records

### 6. Payment Processing

#### Payment Type Handlers
| Type | Handler | Integration |
|------|---------|-------------|
| cash | Direct DB | None |
| card | Direct DB | None |
| app (click) | ClickPass | External API |
| app (payme) | PaymeGo | External API |
| app (uzum) | UzumFastPay | External API |
| app (alif) | AlifPay | External API |

#### Payment Flow
1. Delete existing sale_payments for the sale
2. Process each payment type sequentially
3. For app payments:
   - Retrieve payment service configuration
   - Create sale_payment record
   - Call external payment API
   - Update payment status on success
4. For cash/card payments:
   - Create sale_payment record directly

### 7. Sale Completion
- Update sale status to 'completed'
- Set completion timestamp
- Finalize inventory adjustments

## Data Models

### Sale
```go
type Sale struct {
    ID          string
    Status      string
    CompletedAt *time.Time
    // ... other fields
}
```

### FinalSale (Request)
```go
type FinalSale struct {
    SaleID               string
    StoreID              string
    TotalAmount          float64
    CashBoxOperationId   string
    PaymentTypes         []FinalPaymentType
    MarkingData          interface{}
}
```

### FinalPaymentType
```go
type FinalPaymentType struct {
    Type    string  // cash, card, app
    AppType string  // click, payme, uzum, alif
    Amount  float64
}
```

## Error Handling

### Error Categories
1. **Validation Errors** (400)
   - Invalid JSON format
   - Missing payment types
   - Amount calculation mismatches

2. **Business Logic Errors** (409)
   - Sale already completed
   - Insufficient inventory

3. **System Errors** (500)
   - Database connection issues
   - External payment API failures
   - Transaction commit failures

### Error Response Format
```json
{
  "status": "error",
  "message": "error.code.or.message"
}
```

## Security Considerations

### Authentication
- Bearer token authentication required
- User permissions validated per store access

### Data Integrity
- Database transactions ensure ACID properties
- Concurrent request locking prevents race conditions
- Payment validation before processing

### External API Security
- Secure communication with payment providers
- API key management per store
- Response validation and error handling

## Performance Considerations

### Concurrency
- Mutex locking per sale ID prevents conflicts
- Non-blocking for different sales
- Timeout considerations for long-running operations

### Database Optimization
- Single transaction reduces connection overhead
- Batch operations where possible
- Proper indexing on sale_id and related fields

### External API Management
- Timeout configuration for payment APIs
- Retry logic for transient failures
- Circuit breaker pattern consideration

## Monitoring and Logging

### Key Metrics
- Sale completion success rate
- Payment processing latency
- Error rates by payment type
- Transaction rollback frequency

### Logging Points
- Request initiation and completion
- Payment processing steps
- Error conditions and rollbacks
- External API call results

## Testing Strategy

### Unit Tests
- Payment type validation
- Amount calculation logic
- Error handling scenarios

### Integration Tests
- Database transaction behavior
- External payment API mocking
- End-to-end sale completion flow

### Load Testing
- Concurrent sale processing
- Payment provider integration under load
- Database performance under high transaction volume

## Deployment Considerations

### Configuration
- Payment provider credentials per environment
- Database connection settings
- Timeout and retry configurations

### Rollback Strategy
- Database migration compatibility
- API version management
- Feature flag support for payment types

## Future Enhancements

### Scalability
- Async payment processing consideration
- Event-driven architecture migration
- Microservice decomposition

### Features
- Partial payment support
- Payment installments
- Loyalty program integration
- Advanced inventory reservation

## Dependencies

### Internal Services
- Inventory Service
- Payment Service
- Cashbox Service
- User Authentication Service

### External Services
- Click Payment API
- Payme Payment API
- Uzum Payment API
- Alif Payment API

### Infrastructure
- PostgreSQL Database
- Redis (for locking mechanism)
- Application Load Balancer
- Monitoring and Logging Stack