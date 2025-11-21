# Query Parameter Validation Middleware

The `ValidateQuery` middleware provides type-safe query parameter validation using struct tags and the go-playground/validator library.

## Usage

### 1. Define a Query Parameters Struct

Create a struct with validation tags in your models:

```go
// internal/models/file.go
type FileListQueryParams struct {
    Path      string `json:"path" validate:"max=1024"`
    SortBy    string `json:"sort_by" validate:"omitempty,oneof=name size created_at"`
    SortOrder string `json:"sort_order" validate:"omitempty,oneof=asc desc"`
    Limit     int    `json:"limit" validate:"omitempty,min=1,max=1000"`
    Offset    int    `json:"offset" validate:"omitempty,min=0"`
}
```

### 2. Use the Middleware in Routes

Apply the middleware to your route using Chi's `.With()` method:

```go
// internal/services/file.go
func (s FileService) Routes() chi.Router {
    r := chi.NewRouter()

    r.With(m.ValidateQuery[models.FileListQueryParams]).
        Get("/files", handlers.GetListHandler(s.ListFiles))

    return r
}
```

### 3. Extract Query Parameters in Handler

Use the `GetQueryParams` helper to extract validated parameters:

```go
// In your handler function
func (s FileService) ListFiles(logger *zap.Logger, claims models.UserClaims, ids uuid.UUIDs) ([]models.File, error) {
    queryParams, err := h.GetQueryParams[models.FileListQueryParams](r.Context())
    if err != nil {
        logger.Error("Failed to extract query params from context")
        return nil, customErr.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
    }

    // Use the validated query parameters
    files := s.repository.ListFiles(queryParams.Path, queryParams.SortBy, queryParams.SortOrder, queryParams.Limit, queryParams.Offset)
    return files, nil
}
```

## Supported Field Types

The middleware supports automatic parsing for:
- `string`
- `int`, `int32`, `int64`
- `bool`
- `float32`, `float64`
- Pointer types (`*string`, `*int`, etc.)

## Validation Tags

All standard go-playground/validator tags are supported:

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must be present | `validate:"required"` |
| `omitempty` | Skip validation if empty | `validate:"omitempty,min=1"` |
| `min=N` | Minimum value/length | `validate:"min=1"` |
| `max=N` | Maximum value/length | `validate:"max=100"` |
| `oneof=a b c` | Value must be one of listed | `validate:"oneof=asc desc"` |
| `email` | Valid email format | `validate:"email"` |
| `url` | Valid URL format | `validate:"url"` |
| `uuid` | Valid UUID format | `validate:"uuid"` |

For a complete list, see: https://pkg.go.dev/github.com/go-playground/validator/v10

## Error Handling

The middleware returns `400 Bad Request` with an array of error messages if:
- Query parameters cannot be parsed (invalid type conversion)
- Validation fails based on struct tags

Error response format:
```json
{
    "status": 400,
    "error": [
        "Key: 'FileListQueryParams.Limit' Error:Field validation for 'Limit' failed on the 'max' tag",
        "Key: 'FileListQueryParams.SortBy' Error:Field validation for 'SortBy' failed on the 'oneof' tag"
    ]
}
```

## Example Request

```bash
# Valid request
curl "http://localhost:8080/api/files?path=/documents&sort_by=name&sort_order=asc&limit=50&offset=0"

# Invalid request (limit exceeds max)
curl "http://localhost:8080/api/files?limit=5000"
# Returns: 400 Bad Request with validation error

# Invalid request (invalid sort_by value)
curl "http://localhost:8080/api/files?sort_by=invalid_field"
# Returns: 400 Bad Request with validation error
```

## Implementation Details

- Query parameter names are matched to struct fields using the `json` tag
- Uses reflection for type-safe parsing
- Stores validated data in request context with `models.QueryKey{}`
- Follows the same pattern as the body validation middleware (`Validate`)
- No custom validators are registered (uses standard validators only)
