package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

type apiError struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrInvalidArgument):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrProjectNotFound), errors.Is(err, model.ErrRunNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrGeometryFailed):
		status = http.StatusUnprocessableEntity
	}
	WriteJSON(w, status, apiError{Error: err.Error()})
}
