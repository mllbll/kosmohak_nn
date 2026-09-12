package v1

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func decodeScenario(r io.Reader) (model.Scenario, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return model.Scenario{}, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err)
	}

	var head struct {
		SchemaVersion     string          `json:"schema_version"`
		EffectiveScenario json.RawMessage `json:"effective_scenario"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return model.Scenario{}, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err)
	}

	if head.SchemaVersion == model.ResultSchemaVersion {
		if len(head.EffectiveScenario) == 0 || string(head.EffectiveScenario) == "null" {
			return model.Scenario{}, fmt.Errorf("%w: effective_scenario required for %s", model.ErrInvalidArgument, model.ResultSchemaVersion)
		}
		var sc model.Scenario
		if err := json.Unmarshal(head.EffectiveScenario, &sc); err != nil {
			return model.Scenario{}, fmt.Errorf("%w: effective_scenario: %v", model.ErrInvalidArgument, err)
		}
		return sc, nil
	}

	var sc model.Scenario
	if err := json.Unmarshal(data, &sc); err != nil {
		return model.Scenario{}, fmt.Errorf("%w: %v", model.ErrInvalidArgument, err)
	}
	return sc, nil
}
