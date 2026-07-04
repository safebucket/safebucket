package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/rbac"
	"go.uber.org/zap"

	apierrors "github.com/safebucket/safebucket/internal/errors"
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
			h.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeBadRequest})
			return
		}

		validate := validator.New()
		_ = validate.RegisterValidation("activity_action", func(fl validator.FieldLevel) bool {
			return slices.Contains(activity.ValidActions, fl.Field().String())
		})
		_ = validate.RegisterValidation("rbac_resource", func(fl validator.FieldLevel) bool {
			return slices.Contains(rbac.ValidResources, fl.Field().String())
		})

		err = validate.Struct(data)
		if err != nil {
			var validationErrors validator.ValidationErrors
			if !errors.As(err, &validationErrors) {
				h.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeInvalidRequest})
				return
			}
			codes := make([]string, 0, len(validationErrors))
			for _, fe := range validationErrors {
				codes = append(codes, validationErrorCode(fe))
			}
			h.RespondWithError(w, http.StatusBadRequest, codes)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, models.QueryKey{}, *data)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// parseQueryParams uses reflection to parse URL query parameters into a struct.
// It supports string, int, int32, int64, bool, time.Time (RFC 3339), pointer, and []string types.
// []string fields accept repeated params (?k=a&k=b) and/or comma-separated values (?k=a,b).
func parseQueryParams(queryParams url.Values, data interface{}) error {
	return parseStructFields(queryParams, reflect.ValueOf(data).Elem())
}

func parseStructFields(queryParams url.Values, val reflect.Value) error {
	typ := val.Type()

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if fieldType.Anonymous && field.Kind() == reflect.Struct {
			if err := parseStructFields(queryParams, field); err != nil {
				return err
			}
			continue
		}

		queryParamName := fieldType.Tag.Get("json")
		if queryParamName == "" {
			queryParamName = fieldType.Name
		}

		values, present := queryParams[queryParamName]
		if !present || len(values) == 0 {
			continue
		}

		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String {
			if err := setStringSliceField(field, values); err != nil {
				return err
			}
			continue
		}

		if values[0] == "" {
			continue
		}

		if err := setFieldValue(field, values[0]); err != nil {
			return err
		}
	}

	return nil
}

func setStringSliceField(field reflect.Value, values []string) error {
	if !field.CanSet() {
		return nil
	}

	expanded := make([]string, 0, len(values))
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				expanded = append(expanded, part)
			}
		}
	}

	field.Set(reflect.ValueOf(expanded))
	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	if field.Kind() == reflect.Pointer {
		if !field.CanSet() {
			return nil
		}
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

	if field.Type() == reflect.TypeOf(time.Time{}) {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(parsed.UTC()))
		return nil
	}

	switch field.Kind() { //nolint:exhaustive // only a subset of types is supported
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
		return nil
	}

	return nil
}
