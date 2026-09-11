package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) Create(w http.ResponseWriter, r *http.Request) {
	resp, err := a.runService.Create(r.Context(), model.CreateRunRequest{
		ProjectID: chi.URLParam(r, "id"),
	})
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusCreated, resp)
}
