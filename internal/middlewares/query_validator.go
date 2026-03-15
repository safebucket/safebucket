package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
)

func ValidateQuery[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := new(T)
		queryParams := r.URL.Query()

		err := parseQueryParams(queryParams, data)
		if err != nil {
			zap.L().Error("failed to parse query parameters", zap.Error(err))
			h.RespondWithError(w, http.StatusBadRequest, []string{"BAD_REQUEST", err.Error()})
			return
		}

		validate := validator.New()
		err = validate.Struct(data)
		if err != nil {
			var strErrors []string
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				for _, validationErr := range validationErrors {
					strErrors = append(strErrors, validationErr.Error())
				}
			} else {
				strErrors = append(strErrors, err.Error())
			}
			h.RespondWithError(w, http.StatusBadRequest, strErrors)
			return
		}

		// Store validated query parameters in context
		ctx := r.Context()
		ctx = context.WithValue(ctx, models.QueryKey{}, *data)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// parseQueryParams uses reflection to parse URL query parameters into a struct.
// It supports string, int, int32, int64, bool, and pointer types.
func parseQueryParams(queryParams url.Values, data interface{}) error {
	val := reflect.ValueOf(data).Elem()
	typ := val.Type()

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		queryParamName := fieldType.Tag.Get("json")
		if queryParamName == "" {
			queryParamName = fieldType.Name
		}

		queryValue := queryParams.Get(queryParamName)
		if queryValue == "" {
			// Skip empty values, validation will handle required fields
			continue
		}

		// Set the field value based on its type
		if err := setFieldValue(field, queryValue); err != nil {
			return err
		}
	}

	return nil
}

// setFieldValue sets a struct field value from a string, handling type conversion.
func setFieldValue(field reflect.Value, value string) error {
	if field.Kind() == reflect.Ptr {
		if !field.CanSet() {
			return nil
		}
		// Create a new value of the pointer's element type
		newValue := reflect.New(field.Type().Elem())
		if err := setFieldValue(newValue.Elem(), value); err != nil {
			return err
		}
		field.Set(newValue)
		return nil
	}

	if !field.CanSet() {
		return nil
	}

	switch field.Kind() { //nolint:exhaustive // Only handle supported types.
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	default:
		// Unsupported type, skip
		return nil
	}

	return nil
}
