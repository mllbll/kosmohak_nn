package run

import (
	"context"
	"testing"

	"github.com/mllbll/kosmohak_nn/internal/model"
	runrepo "github.com/mllbll/kosmohak_nn/internal/repository/run"
	"github.com/stretchr/testify/require"
)

func TestCompareRejectsIncompatibleExperiments(t *testing.T) {
	cases := map[string]func(*model.Run){
		"horizon":         func(r *model.Run) { r.EffectiveScenario.Environment.HorizonS *= 2 },
		"step":            func(r *model.Run) { r.EffectiveScenario.Environment.StepS /= 2 },
		"target":          func(r *model.Run) { r.EffectiveScenario.Environment.TargetAvailability = 0.5 },
		"client identity": func(r *model.Run) { r.EffectiveScenario.GroundSites[0].ID = "OTHER" },
		"client location": func(r *model.Run) { r.EffectiveScenario.GroundSites[0].LatDeg++ },
		"extra client": func(r *model.Run) {
			r.EffectiveScenario.GroundSites = append(r.EffectiveScenario.GroundSites, model.GroundSite{ID: "EXTRA", Role: "client"})
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			a, b := testRun("project"), testRun("project")
			change(&b)
			repo := runrepo.NewRepository()
			require.NoError(t, repo.Create(context.Background(), a))
			require.NoError(t, repo.Create(context.Background(), b))
			service := NewService(nil, repo, nil)
			_, err := service.Compare(context.Background(), model.CompareRunsRequest{RunAID: a.ID, RunBID: b.ID})
			require.ErrorIs(t, err, model.ErrInvalidArgument)
		})
	}
}

func TestCompareAllowsDifferentLinkConditionsAndDisplayNames(t *testing.T) {
	a, b := testRun("project"), testRun("project")
	b.EffectiveScenario.Environment.ISLRangeKM = 2000
	b.EffectiveScenario.GroundSites[0].Name = "Новое отображаемое имя"
	b.EffectiveScenario.GroundSites[0], b.EffectiveScenario.GroundSites[1] = b.EffectiveScenario.GroundSites[1], b.EffectiveScenario.GroundSites[0]
	repo := runrepo.NewRepository()
	require.NoError(t, repo.Create(context.Background(), a))
	require.NoError(t, repo.Create(context.Background(), b))
	service := NewService(nil, repo, nil)
	response, err := service.Compare(context.Background(), model.CompareRunsRequest{RunAID: a.ID, RunBID: b.ID})
	require.NoError(t, err)
	require.Contains(t, response.Config, "isl_range_km")
}

func TestCompareRejectsRepeatedRunIDs(t *testing.T) {
	_, err := compareRunIDs(model.CompareRunsRequest{RunIDs: []string{"one", "one"}})
	require.ErrorIs(t, err, model.ErrInvalidArgument)
	_, err = compareRunIDs(model.CompareRunsRequest{RunAID: "one", RunBID: "one"})
	require.ErrorIs(t, err, model.ErrInvalidArgument)
}

func TestGapReasonIsAbsentWhenThereIsNoUniqueDominantReason(t *testing.T) {
	for i := 0; i < 100; i++ {
		require.Equal(t, model.GapNone, dominantReasonCount(map[model.GapReason]int{
			model.GapNoVisibleSat: 2, model.GapISLPartition: 2,
		}))
		require.Equal(t, model.GapGatewayOutage, dominantReasonCount(map[model.GapReason]int{
			model.GapNoVisibleSat: 2, model.GapISLPartition: 2, model.GapGatewayOutage: 3,
		}))
	}
}
