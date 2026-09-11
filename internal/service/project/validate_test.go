package project

import (
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestApplyPatchDoesNotMutateOriginal() {
	var (
		sc = testScenario()
		launchStage = 1
	)

	patched, err := ApplyPatch(sc, model.Patch{LaunchStage: &launchStage})

	s.Require().NoError(err)
	s.Require().Equal(3, sc.Design.LaunchStage)
	s.Require().Equal(launchStage, patched.Design.LaunchStage)
}
