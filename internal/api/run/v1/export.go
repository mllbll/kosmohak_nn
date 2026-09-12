package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) Export(w http.ResponseWriter, r *http.Request) {
	resp, err := a.runService.Export(r.Context(), model.ExportRunRequest{
		RunID: chi.URLParam(r, "id"),
	})
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="cosmo-A-result.json"`)
	apiresp.WriteJSON(w, http.StatusOK, resp)
}
