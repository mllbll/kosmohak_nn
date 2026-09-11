package v1

import "github.com/mllbll/kosmohak_nn/internal/service"

type api struct {
	runService service.RunService
}

func NewAPI(runService service.RunService) *api {
	return &api{
		runService: runService,
	}
}
