package geometry

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func TestExtractPythonError(t *testing.T) {
	stderr := []byte(`Traceback (most recent call last):
  File "runner.py", line 12, in main
    scenario = load(sys.argv[1])
  File "geometry.py", line 11, in load
    validate(scenario)
  File "geometry.py", line 58, in validate
    raise ValueError('Invalid outage')
ValueError: Invalid outage
`)
	got := extractPythonError(stderr)
	if got != "Invalid outage" {
		t.Fatalf("got %q", got)
	}

	err := wrapExec(&exec.ExitError{Stderr: stderr})
	if !errors.Is(err, model.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}

	typeErr := wrapExec(&exec.ExitError{Stderr: []byte("TypeError: 'NoneType' object is not iterable\n")})
	if !errors.Is(typeErr, model.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for TypeError, got %v", typeErr)
	}
}
