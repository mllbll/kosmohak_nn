package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) Patch(w http.ResponseWriter, r *http.Request) {
	var patch model.Patch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		apiresp.WriteError(w, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err))
		return
	}

	resp, err := a.projectService.Patch(r.Context(), model.PatchProjectRequest{
		ProjectID: chi.URLParam(r, "id"),
		Patch:     patch,
	})
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusOK, resp.Project)
}
