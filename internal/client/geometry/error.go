package geometry

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func wrapExec(err error) error {
	if ee, ok := err.(*exec.ExitError); ok {
		msg := extractPythonError(ee.Stderr)
		if isValidationError(ee.Stderr) {
			return fmt.Errorf("%w: %s", model.ErrInvalidArgument, msg)
		}
		return fmt.Errorf("%w: %s", model.ErrGeometryFailed, msg)
	}
	return fmt.Errorf("%w: %v", model.ErrGeometryFailed, err)
}

func isValidationError(stderr []byte) bool {
	text := string(stderr)
	return strings.Contains(text, "ValueError:") ||
		strings.Contains(text, "JSONDecodeError:") ||
		strings.Contains(text, "json.decoder.JSONDecodeError:")
}

func extractPythonError(stderr []byte) string {
	text := strings.TrimSpace(string(stderr))
	if text == "" {
		return "python geometry failed"
	}

	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		for _, prefix := range []string{
			"ValueError: ",
			"json.decoder.JSONDecodeError: ",
			"JSONDecodeError: ",
		} {
			if strings.HasPrefix(line, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}

	return string(bytes.TrimSpace(stderr))
}
