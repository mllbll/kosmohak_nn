package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) Compare(w http.ResponseWriter, r *http.Request) {
	var req model.CompareRunsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresp.WriteError(w, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err))
		return
	}

	resp, err := a.runService.Compare(r.Context(), req)
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusOK, resp)
}
