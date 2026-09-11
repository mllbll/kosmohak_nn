package project

import (
	"sync"

	def "github.com/mllbll/kosmohak_nn/internal/repository"
	repoModel "github.com/mllbll/kosmohak_nn/internal/repository/model"
)

var _ def.ProjectRepository = (*repository)(nil)

type repository struct {
	mu   sync.RWMutex
	data map[string]repoModel.Project
}

func NewRepository() *repository {
	return &repository{
		data: make(map[string]repoModel.Project),
	}
}
