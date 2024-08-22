package common

import (
	"context"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"net/http"
)

func Validate[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := new(T)
		err := json.NewDecoder(r.Body).Decode(&data)
		validate := validator.New()

		err = validate.Struct(data)
		if err != nil {
			var strErrors []string
			for _, err := range err.(validator.ValidationErrors) {
				strErrors = append(strErrors, err.Error())
			}
			RespondWithError(w, 400, strErrors)
			return
		}

		ctx := context.WithValue(r.Context(), "body", *data)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
