package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	h "api/internal/helpers"

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

func validateFilename(fl validator.FieldLevel) bool {
	filename := fl.Field().String()

	// Must have an extension (at least one dot followed by 1-10 alphanumeric chars)
	// Filename part can contain letters, numbers, spaces, underscores, hyphens, dots, and parentheses
	// Must start with alphanumeric or underscore (not a dot or special char)
	regex := regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_ \-.()\[\]]*\.[a-zA-Z0-9]{1,10}$`)

	// Block prohibited characters: / \ < > : " | ? * and null byte
	prohibited := regexp.MustCompile(`[/\\<>:"|?*\x00]`)

	return regex.MatchString(filename) && !prohibited.MatchString(filename)
}

func validateFoldername(fl validator.FieldLevel) bool {
	foldername := fl.Field().String()

	// Folders can contain letters, numbers, spaces, underscores, and hyphens
	// Must start with alphanumeric or underscore (not a space or special char)
	// No extension allowed, no path separators
	regex := regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_ \-]*$`)

	// Block prohibited characters: / \ < > : " | ? * and null byte
	prohibited := regexp.MustCompile(`[/\\<>:"|?*\x00]`)

	return regex.MatchString(foldername) && !prohibited.MatchString(foldername)
}

func Validate[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit

		data := new(T)
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			zap.L().Error("failed to decode body", zap.Error(err))
			h.RespondWithError(w, http.StatusBadRequest, []string{"BAD_REQUEST"})
			return
		}

		validate := validator.New()
		_ = validate.RegisterValidation("filename", validateFilename)
		_ = validate.RegisterValidation("foldername", validateFoldername)
		_ = validate.RegisterValidation("maxuploadsize", validateMaxUploadSize)

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
