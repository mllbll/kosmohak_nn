package project

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestPatchSuccess() {
	var (
		projectID   = gofakeit.UUID()
		sc          = testScenario()
		launchStage = 1

		patchProjectRequest = model.PatchProjectRequest{
			ProjectID: projectID,
			Patch: model.Patch{
				LaunchStage: &launchStage,
			},
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, nil)
	s.projectRepository.On("Update", s.ctx, mock.MatchedBy(func(p model.Project) bool {
		return p.ID == projectID && p.Effective.Design.LaunchStage == launchStage && p.Base.Design.LaunchStage == sc.Design.LaunchStage
	})).Return(nil)

	res, err := s.service.Patch(s.ctx, patchProjectRequest)

	s.Require().NoError(err)
	s.Require().Equal(launchStage, res.Project.Effective.Design.LaunchStage)
	s.Require().Equal(sc.Design.LaunchStage, res.Project.Base.Design.LaunchStage)
}

func (s *ServiceSuite) TestPatchNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		patchProjectRequest = model.PatchProjectRequest{
			ProjectID: projectID,
			Patch:     model.Patch{},
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(model.Project{}, model.ErrProjectNotFound)

	res, err := s.service.Patch(s.ctx, patchProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrProjectNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestPatchInvalidArgument() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		patchProjectRequest = model.PatchProjectRequest{
			ProjectID: projectID,
			Patch: model.Patch{
				Planes: []model.PlanePatch{
					{ID: "UNKNOWN"},
				},
			},
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")
	s.projectRepository.AssertNotCalled(s.T(), "Update")

	res, err := s.service.Patch(s.ctx, patchProjectRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestPatchGeometryError() {
	var (
		projectID   = gofakeit.UUID()
		sc          = testScenario()
		launchStage = 1
		geomErr     = gofakeit.Error()

		patchProjectRequest = model.PatchProjectRequest{
			ProjectID: projectID,
			Patch: model.Patch{
				LaunchStage: &launchStage,
			},
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, geomErr)
	s.projectRepository.AssertNotCalled(s.T(), "Update")

	res, err := s.service.Patch(s.ctx, patchProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, geomErr)
	s.Require().Empty(res)
}
