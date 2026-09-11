package v1

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	t, _ := strconv.ParseFloat(r.URL.Query().Get("t_s"), 64)
	resp, err := a.runService.GetSnapshot(r.Context(), model.GetSnapshotRequest{
		RunID:    chi.URLParam(r, "id"),
		TS:       t,
		ClientID: r.URL.Query().Get("client_id"),
	})
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusOK, resp)
}
