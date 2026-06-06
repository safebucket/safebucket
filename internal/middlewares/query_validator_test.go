package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests"

	"github.com/stretchr/testify/assert"
)

type BasicQueryParams struct {
	Name   string `json:"name"   validate:"required"`
	Status string `json:"status" validate:"omitempty,oneof=active inactive"`
	Limit  int    `json:"limit"  validate:"omitempty,min=1,max=100"`
}

type PointerQueryParams struct {
	Name   *string `json:"name"`
	Status *string `json:"status" validate:"omitempty,oneof=uploaded deleted"`
	Limit  *int    `json:"limit"  validate:"omitempty,min=1,max=1000"`
	Active *bool   `json:"active"`
}

type AllTypesQueryParams struct {
	StringField  string   `json:"string_field"`
	IntField     int      `json:"int_field"`
	Int32Field   int32    `json:"int32_field"`
	Int64Field   int64    `json:"int64_field"`
	BoolField    bool     `json:"bool_field"`
	Float32Field float32  `json:"float32_field"`
	Float64Field float64  `json:"float64_field"`
	PtrString    *string  `json:"ptr_string"`
	PtrInt       *int     `json:"ptr_int"`
	PtrBool      *bool    `json:"ptr_bool"`
	PtrFloat     *float64 `json:"ptr_float"`
}

type JSONTagOptionsParams struct {
	Field1 string `json:"field1,omitempty"`
	Field2 string `json:"field2,string"`
	Field3 string `json:"-"`
}

type NoJSONTagParams struct {
	FieldName string
	OtherName int
}

type UnsupportedTypeParams struct {
	ValidField   string
	MapField     map[string]string
	StructField  struct{ Name string }
	UintField    uint
	ComplexField complex64
}

type SliceParams struct {
	SliceField []string
}

func runMiddleware[T any](_ *testing.T, queryString string) (*httptest.ResponseRecorder, context.Context) {
	req := httptest.NewRequest(http.MethodGet, "/test"+queryString, nil)
	recorder := httptest.NewRecorder()

	ctxChan := make(chan context.Context, 1)
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxChan <- r.Context()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	handler := ValidateQuery[T](customHandler)
	handler.ServeHTTP(recorder, req)

	// Only read from channel if handler was called (status 2xx or 3xx)
	var ctx context.Context
	if recorder.Code >= 200 && recorder.Code < 400 {
		ctx = <-ctxChan
	} else {
		ctx = context.Background()
	}

	return recorder, ctx
}

func TestValidateQueryBasicTypes(t *testing.T) {
	t.Run("Valid basic query params", func(t *testing.T) {
		recorder, ctx := runMiddleware[BasicQueryParams](t, "?name=test&status=active&limit=50")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test", params.Name)
		assert.Equal(t, "active", params.Status)
		assert.Equal(t, 50, params.Limit)
	})

	t.Run("Missing required field", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "?status=active&limit=50")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)

		errors := models.Error{
			Status: http.StatusBadRequest,
			Error: []string{
				"Key: 'BasicQueryParams.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			},
		}
		tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, errors)
	})

	t.Run("Invalid oneof validation", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "?name=test&status=invalid&limit=50")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Integer out of range (max)", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "?name=test&limit=200")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Integer out of range (min)", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "?name=test&limit=0")
		// limit=0 passes because the middleware parses "0" (non-empty) but validator min=1 is not triggered.
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Invalid integer format", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "?name=test&limit=abc")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Empty query params with required field", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Optional fields omitted", func(t *testing.T) {
		recorder, ctx := runMiddleware[BasicQueryParams](t, "?name=test")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test", params.Name)
		assert.Equal(t, "", params.Status)
		assert.Equal(t, 0, params.Limit)
	})
}

func TestValidateQueryPointerTypes(t *testing.T) {
	t.Run("All pointer fields provided", func(t *testing.T) {
		recorder, ctx := runMiddleware[PointerQueryParams](t, "?name=test&status=uploaded&limit=100&active=true")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[PointerQueryParams](ctx)
		assert.NoError(t, err)
		assert.NotNil(t, params.Name)
		assert.Equal(t, "test", *params.Name)
		assert.NotNil(t, params.Status)
		assert.Equal(t, "uploaded", *params.Status)
		assert.NotNil(t, params.Limit)
		assert.Equal(t, 100, *params.Limit)
		assert.NotNil(t, params.Active)
		assert.True(t, *params.Active)
	})

	t.Run("No pointer fields provided", func(t *testing.T) {
		recorder, ctx := runMiddleware[PointerQueryParams](t, "")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[PointerQueryParams](ctx)
		assert.NoError(t, err)
		assert.Nil(t, params.Name)
		assert.Nil(t, params.Status)
		assert.Nil(t, params.Limit)
		assert.Nil(t, params.Active)
	})

	t.Run("Partial pointer fields", func(t *testing.T) {
		recorder, ctx := runMiddleware[PointerQueryParams](t, "?name=test&limit=50")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[PointerQueryParams](ctx)
		assert.NoError(t, err)
		assert.NotNil(t, params.Name)
		assert.Nil(t, params.Status)
		assert.NotNil(t, params.Limit)
		assert.Nil(t, params.Active)
	})

	t.Run("Invalid pointer validation", func(t *testing.T) {
		recorder, _ := runMiddleware[PointerQueryParams](t, "?status=invalid")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestValidateQueryAllTypes(t *testing.T) {
	t.Run("All types valid", func(t *testing.T) {
		queryString := "?string_field=test&int_field=42&int32_field=32&int64_field=64" +
			"&bool_field=true&float32_field=3.14&float64_field=2.718" +
			"&ptr_string=ptr&ptr_int=99&ptr_bool=false&ptr_float=1.23"

		recorder, ctx := runMiddleware[AllTypesQueryParams](t, queryString)
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test", params.StringField)
		assert.Equal(t, 42, params.IntField)
		assert.Equal(t, int32(32), params.Int32Field)
		assert.Equal(t, int64(64), params.Int64Field)
		assert.True(t, params.BoolField)
		assert.InDelta(t, 3.14, params.Float32Field, 0.01)
		assert.InDelta(t, 2.718, params.Float64Field, 0.001)
		assert.NotNil(t, params.PtrString)
		assert.Equal(t, "ptr", *params.PtrString)
		assert.NotNil(t, params.PtrInt)
		assert.Equal(t, 99, *params.PtrInt)
		assert.NotNil(t, params.PtrBool)
		assert.False(t, *params.PtrBool)
		assert.NotNil(t, params.PtrFloat)
		assert.InDelta(t, 1.23, *params.PtrFloat, 0.001)
	})

	t.Run("Boolean variations", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?bool_field=1&ptr_bool=0")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.True(t, params.BoolField)
		assert.NotNil(t, params.PtrBool)
		assert.False(t, *params.PtrBool)
	})

	t.Run("Negative numbers", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](
			t,
			"?int_field=-42&int32_field=-32&int64_field=-64&float64_field=-3.14",
		)
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, -42, params.IntField)
		assert.Equal(t, int32(-32), params.Int32Field)
		assert.Equal(t, int64(-64), params.Int64Field)
		assert.InDelta(t, -3.14, params.Float64Field, 0.001)
	})

	t.Run("Zero values", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?int_field=0&bool_field=false&float64_field=0.0")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, params.IntField)
		assert.False(t, params.BoolField)
		assert.Equal(t, 0.0, params.Float64Field)
	})
}

func TestValidateQueryInvalidTypes(t *testing.T) {
	t.Run("Invalid boolean", func(t *testing.T) {
		recorder, _ := runMiddleware[AllTypesQueryParams](t, "?bool_field=notabool")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Invalid float", func(t *testing.T) {
		recorder, _ := runMiddleware[AllTypesQueryParams](t, "?float64_field=notafloat")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Float overflow", func(t *testing.T) {
		recorder, _ := runMiddleware[AllTypesQueryParams](t, "?float64_field=1.7976931348623159e+309")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestValidateQueryJSONTagParsing(t *testing.T) {
	t.Run("JSON tag with options - BUG", func(t *testing.T) {
		recorder, ctx := runMiddleware[JSONTagOptionsParams](t, "?field1=value1&field2=value2")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[JSONTagOptionsParams](ctx)
		assert.NoError(t, err)
		t.Logf("Note: JSON tag options are not properly handled. Params: %+v", params)
	})

	t.Run("No JSON tag uses field name", func(t *testing.T) {
		recorder, ctx := runMiddleware[NoJSONTagParams](t, "?FieldName=value1&OtherName=42")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[NoJSONTagParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "value1", params.FieldName)
		assert.Equal(t, 42, params.OtherName)
	})
}

func TestValidateQueryUnsupportedTypes(t *testing.T) {
	t.Run("Unsupported types are silently skipped", func(t *testing.T) {
		recorder, ctx := runMiddleware[UnsupportedTypeParams](t,
			"?ValidField=test&MapField=key:value&UintField=42&ComplexField=1+2i")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[UnsupportedTypeParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test", params.ValidField)
		assert.Nil(t, params.MapField)
		assert.Equal(t, "", params.StructField.Name)
		assert.Equal(t, uint(0), params.UintField)
		assert.Equal(t, complex64(0), params.ComplexField)
	})
}

func TestValidateQueryStringSlice(t *testing.T) {
	t.Run("Comma-separated values", func(t *testing.T) {
		recorder, ctx := runMiddleware[SliceParams](t, "?SliceField=a,b,c")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[SliceParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, params.SliceField)
	})

	t.Run("Repeated params", func(t *testing.T) {
		recorder, ctx := runMiddleware[SliceParams](t, "?SliceField=a&SliceField=b&SliceField=c,d")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[SliceParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c", "d"}, params.SliceField)
	})
}

func TestValidateQueryEdgeCases(t *testing.T) {
	t.Run("Empty string values are skipped", func(t *testing.T) {
		recorder, _ := runMiddleware[BasicQueryParams](t, "?name=&status=&limit=")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Special characters in string", func(t *testing.T) {
		recorder, ctx := runMiddleware[BasicQueryParams](t, "?name=test%20value&status=active")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test value", params.Name)
	})

	t.Run("Unicode in string", func(t *testing.T) {
		recorder, ctx := runMiddleware[BasicQueryParams](t, "?name=%E4%BD%A0%E5%A5%BD&status=active")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "你好", params.Name)
	})

	t.Run("Very large integer (within int64 range)", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?int64_field=9223372036854775807")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(9223372036854775807), params.Int64Field)
	})

	t.Run("Scientific notation for float", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?float64_field=1.23e-4")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.InDelta(t, 0.000123, params.Float64Field, 0.0000001)
	})
}

func TestValidateQueryMultipleValues(t *testing.T) {
	t.Run("Multiple values for same parameter - BUG", func(t *testing.T) {
		recorder, ctx := runMiddleware[PointerQueryParams](t, "?status=uploaded&status=deleted")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[PointerQueryParams](ctx)
		assert.NoError(t, err)
		assert.NotNil(t, params.Status)
		// BUG: Only the first value is used because queryParams.Get() returns first value
		assert.Equal(t, "uploaded", *params.Status)
		t.Log("Note: Multiple query parameters with same name - only first value is used")
	})
}

func TestValidateQueryActivityParams(t *testing.T) {
	t.Run("Valid range, cursor and limit", func(t *testing.T) {
		recorder, ctx := runMiddleware[models.ActivityQueryParams](
			t, "?from=2026-05-01T00:00:00Z&to=2026-06-01T00:00:00Z&cursor=1720000000000000000&limit=50",
		)
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[models.ActivityQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), params.From)
		assert.Equal(t, "1720000000000000000", params.Cursor)
		assert.Equal(t, 50, params.Limit)
	})

	t.Run("Empty range is allowed", func(t *testing.T) {
		recorder, _ := runMiddleware[models.ActivityQueryParams](t, "")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Malformed from is rejected", func(t *testing.T) {
		recorder, _ := runMiddleware[models.ActivityQueryParams](t, "?from=2026-05-01")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Limit above max is rejected", func(t *testing.T) {
		recorder, _ := runMiddleware[models.ActivityQueryParams](t, "?limit=500")
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestValidateQueryContextStorage(t *testing.T) {
	t.Run("Query params stored in context", func(t *testing.T) {
		recorder, ctx := runMiddleware[BasicQueryParams](t, "?name=test&status=active&limit=50")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test", params.Name)

		value := ctx.Value(models.QueryKey{})
		assert.NotNil(t, value)
	})

	t.Run("Invalid type assertion returns error", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), models.QueryKey{}, "wrong type")

		_, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.Error(t, err)
		assert.Equal(t, "invalid query params", err.Error())
	})

	t.Run("Missing query key returns error", func(t *testing.T) {
		ctx := context.Background()

		_, err := helpers.GetQueryParams[BasicQueryParams](ctx)
		assert.Error(t, err)
		assert.Equal(t, "invalid query params", err.Error())
	})
}

func TestValidateQueryInt32Overflow(t *testing.T) {
	t.Run("Int32 max value", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?int32_field=2147483647")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, int32(2147483647), params.Int32Field)
	})

	t.Run("Int32 min value", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?int32_field=-2147483648")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		assert.Equal(t, int32(-2147483648), params.Int32Field)
	})

	t.Run("Int32 overflow (above max) - BUG", func(t *testing.T) {
		recorder, ctx := runMiddleware[AllTypesQueryParams](t, "?int32_field=2147483648")
		assert.Equal(t, http.StatusOK, recorder.Code)

		params, err := helpers.GetQueryParams[AllTypesQueryParams](ctx)
		assert.NoError(t, err)
		t.Logf("Note: Int32 overflow not detected. Value: %d", params.Int32Field)
	})
}

func TestValidateQueryBoolVariations(t *testing.T) {
	validBoolValues := []string{"1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"}

	for _, value := range validBoolValues {
		t.Run("Bool value: "+value, func(t *testing.T) {
			recorder, _ := runMiddleware[AllTypesQueryParams](t, "?bool_field="+value)
			assert.Equal(t, http.StatusOK, recorder.Code)
			t.Logf("Note: Boolean parsing accepts many variations: %s", value)
		})
	}
}

func TestValidateQueryPerformance(t *testing.T) {
	t.Run("Validator instance created per request - Performance Issue", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			recorder, _ := runMiddleware[BasicQueryParams](t, "?name=test&status=active")
			assert.Equal(t, http.StatusOK, recorder.Code)
		}

		t.Log("Performance Issue: New validator instance created for each request")
	})
}

func BenchmarkValidateQueryMiddleware(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test?name=test&status=active&limit=50", nil)
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := ValidateQuery[BasicQueryParams](customHandler)

	b.ResetTimer()
	for range b.N {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}
}

func BenchmarkValidateQueryWithPointers(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test?name=test&status=uploaded&limit=100&active=true", nil)
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := ValidateQuery[PointerQueryParams](customHandler)

	b.ResetTimer()
	for range b.N {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}
}
