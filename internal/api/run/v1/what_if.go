package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (a *api) WhatIf(w http.ResponseWriter, r *http.Request) {
	var req model.WhatIfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresp.WriteError(w, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err))
		return
	}
	req.RunID = chi.URLParam(r, "id")

	resp, err := a.runService.WhatIf(r.Context(), req)
	if err != nil {
		apiresp.WriteError(w, err)
		return
	}

	apiresp.WriteJSON(w, http.StatusCreated, resp)
}
