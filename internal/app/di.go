package app

import (
	"github.com/mllbll/kosmohak_nn/internal/client/geometry"
	"github.com/mllbll/kosmohak_nn/internal/config"
	"github.com/mllbll/kosmohak_nn/internal/repository"
	projectRepository "github.com/mllbll/kosmohak_nn/internal/repository/project"
	runRepository "github.com/mllbll/kosmohak_nn/internal/repository/run"
	"github.com/mllbll/kosmohak_nn/internal/service"
	projectService "github.com/mllbll/kosmohak_nn/internal/service/project"
	runService "github.com/mllbll/kosmohak_nn/internal/service/run"
)

type diContainer struct {
	cfg config.Config

	geometryClient *geometry.Client

	projectRepository repository.ProjectRepository
	runRepository     repository.RunRepository

	projectService service.ProjectService
	runService     service.RunService
}

func NewDiContainer(cfg config.Config) *diContainer {
	return &diContainer{cfg: cfg}
}

func (d *diContainer) GeometryClient() *geometry.Client {
	if d.geometryClient == nil {
		d.geometryClient = geometry.NewClient(d.cfg.PythonBin, d.cfg.RunnerScript)
	}
	return d.geometryClient
}

func (d *diContainer) ProjectRepository() repository.ProjectRepository {
	if d.projectRepository == nil {
		d.projectRepository = projectRepository.NewRepository()
	}
	return d.projectRepository
}

func (d *diContainer) RunRepository() repository.RunRepository {
	if d.runRepository == nil {
		d.runRepository = runRepository.NewRepository()
	}
	return d.runRepository
}

func (d *diContainer) ProjectService() service.ProjectService {
	if d.projectService == nil {
		d.projectService = projectService.NewService(d.ProjectRepository(), d.GeometryClient())
	}
	return d.projectService
}

func (d *diContainer) RunService() service.RunService {
	if d.runService == nil {
		d.runService = runService.NewService(d.ProjectRepository(), d.RunRepository(), d.GeometryClient())
	}
	return d.runService
}
