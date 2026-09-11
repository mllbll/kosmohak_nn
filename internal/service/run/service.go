package run

import (
	"github.com/mllbll/kosmohak_nn/internal/client/geometry"
	"github.com/mllbll/kosmohak_nn/internal/repository"
	def "github.com/mllbll/kosmohak_nn/internal/service"
)

var _ def.RunService = (*service)(nil)

type service struct {
	projectRepository repository.ProjectRepository
	runRepository     repository.RunRepository
	geometryClient    geometry.Client
}

func NewService(
	projectRepository repository.ProjectRepository,
	runRepository repository.RunRepository,
	geometryClient geometry.Client,
) *service {
	return &service{
		projectRepository: projectRepository,
		runRepository:     runRepository,
		geometryClient:    geometryClient,
	}
}
