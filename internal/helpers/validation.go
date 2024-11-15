package helpers

import (
	"context"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"net/http"
	"regexp"
)

type BodyKey struct{}

func validateFilename(fl validator.FieldLevel) bool {
	fileType := fl.Parent().FieldByName("Type").String()
	if fileType == "file" {
		regex := regexp.MustCompile(`^[a-zA-Z0-9_\-]+\.[a-zA-Z0-9]+$`)
		return regex.MatchString(fl.Field().String())
	}
	return true
}

func Validate[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := new(T)
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			zap.L().Error("failed to decode body", zap.Error(err))
			RespondWithError(w, 400, []string{"failed to decode body"})
			return
		}

		validate := validator.New()
		_ = validate.RegisterValidation("filename", validateFilename)

		err = validate.Struct(data)
		if err != nil {
			var strErrors []string
			for _, err := range err.(validator.ValidationErrors) {
				strErrors = append(strErrors, err.Error())
			}
			RespondWithError(w, 400, strErrors)
			return
		}

		ctx := context.WithValue(r.Context(), BodyKey{}, *data)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
