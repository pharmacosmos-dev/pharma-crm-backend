# Error Handling Guide

## Overview

This guide explains the proper error handling pattern implemented in the application.

## Architecture

### Service Layer (domain/error_types.go)
Services return `domain.Error` types which contain:
- `Code`: HTTP status code (400, 404, 500, etc.)
- `Message`: Error message key for i18n

Example:
```go
var (
    NotFoundError = NewError(http.StatusNotFound, "not.found")
    InvalidRequestBodyError = NewError(http.StatusBadRequest, "invalid.request.body")
    InternalServerError = NewError(http.StatusInternalServerError, "internal.server.error")
)
```

### Handler Layer (internal/controller/http/v1/)
Handlers use `handleServiceResponse()` to automatically handle service responses.

## Usage Pattern

### ✅ CORRECT - Using handleServiceResponse

```go
func (h *DraftHandler) Get(c *gin.Context) {
    id := c.Param("id")
    
    ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
    defer cancel()
    
    draft, err := h.service.GetDraftById(ctx, id)
    handleServiceResponse(c, draft, err)  // ✅ Automatically handles error codes
}
```

### ❌ INCORRECT - Manual error handling

```go
func (h *DraftHandler) Get(c *gin.Context) {
    id := c.Param("id")
    
    ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
    defer cancel()
    
    draft, err := h.service.GetDraftById(ctx, id)
    if err != nil {
        handleResponse(c, InternalError, err.Error())  // ❌ Always returns 500
        return
    }
    
    handleResponse(c, OK, draft)
}
```

## Benefits

1. **Automatic Status Code Mapping**: Service errors with 404 code return 404, not 500
2. **Consistent Error Format**: All errors follow the same response structure
3. **Less Boilerplate**: No need to check error types in every handler
4. **Type Safety**: Compile-time checking of error types

## Service Layer Best Practices

### Return domain.Error for known errors:
```go
func (s *Services) GetDraftById(ctx context.Context, id string) (*domain.Draft, error) {
    var draft domain.Draft
    err := s.db.First(&draft, "id = ?", id).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, domain.NotFoundError  // ✅ Returns 404
        }
        s.log.Errorf("could not get draft: %v", err)
        return nil, domain.InternalServerError  // ✅ Returns 500
    }
    return &draft, nil
}
```

### ❌ Don't return raw errors:
```go
func (s *Services) GetDraftById(ctx context.Context, id string) (*domain.Draft, error) {
    var draft domain.Draft
    err := s.db.First(&draft, "id = ?", id).Error
    if err != nil {
        return nil, err  // ❌ Handler won't know the proper status code
    }
    return &draft, nil
}
```

## Handler Layer Best Practices

### For validation errors (before calling service):
```go
if err := c.ShouldBindJSON(&body); err != nil {
    handleResponse(c, BadRequest, domain.InvalidRequestBodyError.Message)
    return
}
```

### For service calls:
```go
res, err := h.service.CreateDraft(ctx, &body)
handleServiceResponse(c, res, err)  // ✅ Handles both success and error
```

### For list endpoints with count:
```go
res, totalCount, err := h.service.GetDrafts(ctx, &params)
if err != nil {
    handleServiceResponse(c, nil, err)
    return
}

data := utils.ListResponse(res, totalCount, limit, offset)
handleResponse(c, OK, data)
```

## Migration Checklist

To migrate existing handlers:

1. ✅ Ensure service returns `domain.Error` types
2. ✅ Replace manual error handling with `handleServiceResponse()`
3. ✅ Use `.Message` when passing domain errors to `handleResponse()`
4. ✅ Test that proper status codes are returned

## Example Response Formats

### Success (200):
```json
{
  "ok": true,
  "code": 200,
  "message": "The request has succeeded.",
  "data": { ... }
}
```

### Not Found (404):
```json
{
  "ok": false,
  "code": 404,
  "message": "not.found",
  "data": null
}
```

### Internal Error (500):
```json
{
  "ok": false,
  "code": 500,
  "message": "internal.server.error",
  "data": null
}
```
