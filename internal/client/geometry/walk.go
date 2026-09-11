package geometry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (c *Client) WalkSnapshots(ctx context.Context, sc model.Scenario, fn func(model.Snapshot) error) error {
	path, cleanup, err := writeTempScenario(sc)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, c.pythonBin, c.runnerScript, path)
	out, err := cmd.Output()
	if err != nil {
		return wrapExec(err)
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var snap model.Snapshot
		if err := dec.Decode(&snap); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := fn(snap); err != nil {
			return err
		}
	}
}
