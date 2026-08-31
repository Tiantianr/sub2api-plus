package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type weeklyLimitEstimateStoreStub struct {
	snapshotUpdateAccountRepo
}

func (r *weeklyLimitEstimateStoreStub) StoreAccountUsageObservation(ctx context.Context, accountID int64, updates map[string]any) error {
	return r.UpdateExtra(ctx, accountID, updates)
}

func TestObserveOpenAIWeeklyLimitEstimateBaselineAdvanceAndFreeze(t *testing.T) {
	resetAt := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	baselineEstimate, baseline, changed := observeOpenAIWeeklyLimitEstimate(nil, &UsageProgress{
		Utilization: 6,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 150},
	}, now)
	require.True(t, changed)
	require.Nil(t, baselineEstimate)
	require.Equal(t, 6, baseline.LastObservedPercent)
	require.Nil(t, baseline.Estimate)

	estimate, advanced, changed := observeOpenAIWeeklyLimitEstimate(map[string]any{
		openAIWeeklyLimitEstimateExtraKey: baseline,
	}, &UsageProgress{
		Utilization: 7,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 160.87, UserCost: 59.76},
	}, now.Add(time.Hour))
	require.True(t, changed)
	require.NotNil(t, estimate)
	require.InDelta(t, 160.87/6*100, estimate.EstimatedCost, 0.000001)
	require.Equal(t, 160.87, estimate.SampledCost)
	require.Equal(t, 6, estimate.BasisPercent)
	require.Equal(t, 7, estimate.ObservedPercent)
	require.Equal(t, 7, advanced.LastObservedPercent)

	frozen, unchanged, changed := observeOpenAIWeeklyLimitEstimate(map[string]any{
		openAIWeeklyLimitEstimateExtraKey: advanced,
	}, &UsageProgress{
		Utilization: 7,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 220.50},
	}, now.Add(2*time.Hour))
	require.False(t, changed)
	require.Equal(t, advanced, unchanged)
	require.InDelta(t, estimate.EstimatedCost, frozen.EstimatedCost, 0.000001)
	require.Equal(t, 160.87, frozen.SampledCost, "same-percent cost growth must not move the estimate")
}

func TestObserveOpenAIWeeklyLimitEstimateUsesPreviousObservedPercentWhenSkipping(t *testing.T) {
	resetAt := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	snapshot := &openAIWeeklyLimitEstimateSnapshot{
		WindowResetAt:       resetAt.Format(time.RFC3339),
		LastObservedPercent: 6,
	}

	estimate, next, changed := observeOpenAIWeeklyLimitEstimate(map[string]any{
		openAIWeeklyLimitEstimateExtraKey: snapshot,
	}, &UsageProgress{
		Utilization: 8,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 180},
	}, time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC))

	require.True(t, changed)
	require.Equal(t, 6, estimate.BasisPercent)
	require.Equal(t, 8, estimate.ObservedPercent)
	require.InDelta(t, 3000, estimate.EstimatedCost, 0.000001)
	require.Equal(t, 8, next.LastObservedPercent)
}

func TestObserveOpenAIWeeklyLimitEstimateWindowLifecycle(t *testing.T) {
	resetAt := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	previousEstimate := &AccountCostLimitEstimate{
		EstimatedCost:   2500,
		SampledCost:     150,
		BasisPercent:    6,
		ObservedPercent: 7,
		SampledAt:       now.Format(time.RFC3339),
	}

	t.Run("small reset drift stays in the same window", func(t *testing.T) {
		drifted := resetAt.Add(14 * time.Minute)
		estimate, snapshot, changed := observeOpenAIWeeklyLimitEstimate(map[string]any{
			openAIWeeklyLimitEstimateExtraKey: &openAIWeeklyLimitEstimateSnapshot{
				WindowResetAt:       resetAt.Format(time.RFC3339),
				LastObservedPercent: 7,
				Estimate:            previousEstimate,
			},
		}, &UsageProgress{
			Utilization: 8,
			ResetsAt:    &drifted,
			WindowStats: &WindowStats{Cost: 210},
		}, now.Add(time.Hour))

		require.True(t, changed)
		require.Equal(t, 7, estimate.BasisPercent)
		require.Equal(t, 8, snapshot.LastObservedPercent)
	})

	t.Run("new reset anchor clears and re-baselines", func(t *testing.T) {
		newReset := resetAt.Add(7 * 24 * time.Hour)
		estimate, snapshot, changed := observeOpenAIWeeklyLimitEstimate(map[string]any{
			openAIWeeklyLimitEstimateExtraKey: &openAIWeeklyLimitEstimateSnapshot{
				WindowResetAt:       resetAt.Format(time.RFC3339),
				LastObservedPercent: 7,
				Estimate:            previousEstimate,
			},
		}, &UsageProgress{
			Utilization: 1,
			ResetsAt:    &newReset,
			WindowStats: &WindowStats{Cost: 20},
		}, now.Add(7*24*time.Hour))

		require.True(t, changed)
		require.Nil(t, estimate)
		require.Equal(t, 1, snapshot.LastObservedPercent)
		require.Nil(t, snapshot.Estimate)
	})

	t.Run("utilization decrease clears and re-baselines", func(t *testing.T) {
		estimate, snapshot, changed := observeOpenAIWeeklyLimitEstimate(map[string]any{
			openAIWeeklyLimitEstimateExtraKey: &openAIWeeklyLimitEstimateSnapshot{
				WindowResetAt:       resetAt.Format(time.RFC3339),
				LastObservedPercent: 7,
				Estimate:            previousEstimate,
			},
		}, &UsageProgress{
			Utilization: 5,
			ResetsAt:    &resetAt,
			WindowStats: &WindowStats{Cost: 170},
		}, now.Add(2*time.Hour))

		require.True(t, changed)
		require.Nil(t, estimate)
		require.Equal(t, 5, snapshot.LastObservedPercent)
		require.Nil(t, snapshot.Estimate)
	})
}

func TestAttachOpenAIWeeklyLimitEstimatePersistsBaseline(t *testing.T) {
	updates := make(chan map[string]any, 1)
	repo := &weeklyLimitEstimateStoreStub{snapshotUpdateAccountRepo: snapshotUpdateAccountRepo{updateExtraCalls: updates}}
	svc := &AccountUsageService{accountRepo: repo}
	resetAt := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	account := &Account{ID: 42, Extra: map[string]any{}}
	weekly := &UsageProgress{
		Utilization: 7,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 160.87},
	}

	svc.attachOpenAIWeeklyLimitEstimate(context.Background(), account, weekly, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))

	require.Nil(t, weekly.AccountCostLimitEstimate)
	update := <-updates
	snapshot, ok := update[openAIWeeklyLimitEstimateExtraKey].(*openAIWeeklyLimitEstimateSnapshot)
	require.True(t, ok)
	require.Equal(t, 7, snapshot.LastObservedPercent)
	require.Same(t, snapshot, account.Extra[openAIWeeklyLimitEstimateExtraKey])
}

func TestAttachOpenAIWeeklyLimitEstimateFreezesConcurrentStaleObservations(t *testing.T) {
	updates := make(chan map[string]any, 2)
	repo := &weeklyLimitEstimateStoreStub{snapshotUpdateAccountRepo: snapshotUpdateAccountRepo{updateExtraCalls: updates}}
	svc := &AccountUsageService{accountRepo: repo}
	resetAt := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	baseline := &openAIWeeklyLimitEstimateSnapshot{
		WindowResetAt:       resetAt.Format(time.RFC3339),
		LastObservedPercent: 6,
	}
	firstAccount := &Account{ID: 42, Extra: map[string]any{openAIWeeklyLimitEstimateExtraKey: baseline}}
	secondStaleAccount := &Account{ID: 42, Extra: map[string]any{openAIWeeklyLimitEstimateExtraKey: baseline}}
	firstWeekly := &UsageProgress{
		Utilization: 7,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 160.87},
	}
	secondWeekly := &UsageProgress{
		Utilization: 7,
		ResetsAt:    &resetAt,
		WindowStats: &WindowStats{Cost: 220.50},
	}

	svc.attachOpenAIWeeklyLimitEstimate(context.Background(), firstAccount, firstWeekly, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	svc.attachOpenAIWeeklyLimitEstimate(context.Background(), secondStaleAccount, secondWeekly, time.Date(2026, 9, 1, 0, 1, 0, 0, time.UTC))

	require.Len(t, updates, 1)
	require.Equal(t, 160.87, firstWeekly.AccountCostLimitEstimate.SampledCost)
	require.Equal(t, 160.87, secondWeekly.AccountCostLimitEstimate.SampledCost)
}
