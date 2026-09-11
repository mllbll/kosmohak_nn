package service

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

type ProjectService interface {
	Create(ctx context.Context, req model.CreateProjectRequest) (model.CreateProjectResponse, error)
	Get(ctx context.Context, req model.GetProjectRequest) (model.GetProjectResponse, error)
	Patch(ctx context.Context, req model.PatchProjectRequest) (model.PatchProjectResponse, error)
}

type RunService interface {
	Create(ctx context.Context, req model.CreateRunRequest) (model.CreateRunResponse, error)
	Get(ctx context.Context, req model.GetRunRequest) (model.GetRunResponse, error)
	GetSnapshot(ctx context.Context, req model.GetSnapshotRequest) (model.GetSnapshotResponse, error)
	Export(ctx context.Context, req model.ExportRunRequest) (model.ExportDocument, error)
	Compare(ctx context.Context, req model.CompareRunsRequest) (model.CompareRunsResponse, error)
}
