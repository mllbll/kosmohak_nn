package project

import (
	"github.com/mllbll/kosmohak_nn/internal/client/geometry"
	"github.com/mllbll/kosmohak_nn/internal/repository"
	def "github.com/mllbll/kosmohak_nn/internal/service"
)

var _ def.ProjectService = (*service)(nil)

type service struct {
	projectRepository repository.ProjectRepository
	geometryClient    geometry.Client
}

func NewService(
	projectRepository repository.ProjectRepository,
	geometryClient geometry.Client,
) *service {
	return &service{
		projectRepository: projectRepository,
		geometryClient:    geometryClient,
	}
}
