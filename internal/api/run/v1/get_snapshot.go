package v1

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("t_s")
	t := 0.0
	if raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			apiresp.WriteError(w, fmt.Errorf("%w: t_s must be a finite number", model.ErrInvalidArgument))
			return
		}
		t = parsed
	}

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
