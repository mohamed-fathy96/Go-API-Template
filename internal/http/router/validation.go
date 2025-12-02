package router

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"kabsa/internal/http/responses"
	"net/http"
)

var validate = validator.New()

// BindAndValidate reads JSON body into dst and runs validation with tags `validate:"..."`.
func BindAndValidate[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		responses.WriteBadRequest(w, "Invalid JSON payload.")
		return false
	}

	if err := validate.Struct(dst); err != nil {
		responses.WriteBadRequest(w, "Invalid JSON payload.")
		return false
	}

	return true
}
