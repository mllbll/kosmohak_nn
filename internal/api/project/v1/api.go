package v1

import "github.com/mllbll/kosmohak_nn/internal/service"

type api struct {
	projectService service.ProjectService
}

func NewAPI(projectService service.ProjectService) *api {
	return &api{
		projectService: projectService,
	}
}
