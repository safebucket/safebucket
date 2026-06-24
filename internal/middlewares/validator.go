package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type BodyKey struct{}

var maxUploadSize int64

func InitValidator(maxSize int64) {
	maxUploadSize = maxSize
}

func validateMaxUploadSize(fl validator.FieldLevel) bool {
	return fl.Field().Int() <= maxUploadSize
}

var allowedNameChars = regexp.MustCompile(`^[\p{L}\p{M}\p{N} ._()&+#@!~=%$;{}^',\[\]-]+$`)

var reservedNames = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)`)

func validateFilename(fl validator.FieldLevel) bool {
	name := fl.Field().String()

	if name == "" || name == "." || name == ".." {
		return false
	}

	if name != strings.TrimSpace(name) || strings.HasSuffix(name, ".") {
		return false
	}

	return allowedNameChars.MatchString(name) && !reservedNames.MatchString(name)
}

func validateFutureDate(fl validator.FieldLevel) bool {
	t, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return t.After(time.Now())
}

func Validate[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit

		data := new(T)
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			zap.L().Error("failed to decode body", zap.Error(err))
			h.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeBadRequest})
			return
		}

		validate := validator.New()
		_ = validate.RegisterValidation("filename", validateFilename)
		_ = validate.RegisterValidation("foldername", validateFilename)
		_ = validate.RegisterValidation("maxuploadsize", validateMaxUploadSize)
		_ = validate.RegisterValidation("futuredate", validateFutureDate)

		err = validate.Struct(data)
		if err != nil {
			var strErrors []string
			for _, err := range func() validator.ValidationErrors {
				var target validator.ValidationErrors
				_ = errors.As(err, &target)
				return target
			}() {
				strErrors = append(strErrors, err.Error())
			}
			h.RespondWithError(w, http.StatusBadRequest, strErrors)
			return
		}

		ctx := context.WithValue(r.Context(), BodyKey{}, *data)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
