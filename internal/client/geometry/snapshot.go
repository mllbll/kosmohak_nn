package geometry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (c *pythonClient) Snapshot(ctx context.Context, sc model.Scenario, t float64) (model.Snapshot, error) {
	path, cleanup, err := writeTempScenario(sc)
	if err != nil {
		return model.Snapshot{}, err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, c.pythonBin, c.runnerScript, path, fmt.Sprintf("%g", t))
	out, err := cmd.Output()
	if err != nil {
		return model.Snapshot{}, wrapExec(err)
	}

	var snap model.Snapshot
	if err := json.Unmarshal(out, &snap); err != nil {
		return model.Snapshot{}, err
	}
	return snap, nil
}

func writeTempScenario(sc model.Scenario) (string, func(), error) {
	f, err := os.CreateTemp("", "cosmo-scenario-*.json")
	if err != nil {
		return "", nil, err
	}
	if err := json.NewEncoder(f).Encode(model.CloneScenario(sc)); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}
