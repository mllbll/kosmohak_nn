package v1

import (
	"net/http"

	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) Create(w http.ResponseWriter, r *http.Request) {
	sc, err := decodeScenario(r.Body)
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	resp, err := a.projectService.Create(r.Context(), model.CreateProjectRequest{Scenario: sc})
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusCreated, resp.Project)
}
