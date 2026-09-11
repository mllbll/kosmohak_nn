package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) Create(w http.ResponseWriter, r *http.Request) {
	var sc model.Scenario
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		apiresp.WriteError(w, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err))
		return
	}

	resp, err := a.projectService.Create(r.Context(), model.CreateProjectRequest{Scenario: sc})
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusCreated, resp.Project)
}
