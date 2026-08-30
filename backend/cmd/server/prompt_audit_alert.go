package main

import (
	"context"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

const (
	promptAuditPoolFailureThreshold = 5
	promptAuditPoolAlertTimeout     = 45 * time.Second
)

type promptAuditPoolFailureAlert struct {
	mu          sync.Mutex
	consecutive int
	generation  uint64
	notify      func(int, securityaudit.DecisionKind) bool
	alerts      *service.OpsAlertEvaluatorService
}

func providePromptAuditMetrics(alerts *service.OpsAlertEvaluatorService) *securityaudit.AtomicMetrics {
	metrics := securityaudit.NewAtomicMetrics()
	alert := newPromptAuditPoolFailureAlert(alerts)
	metrics.SetOutcomeObserver(alert.Observe)
	return metrics
}

func newPromptAuditPoolFailureAlert(alerts *service.OpsAlertEvaluatorService) *promptAuditPoolFailureAlert {
	alert := &promptAuditPoolFailureAlert{alerts: alerts}
	alert.notify = alert.sendEmail
	return alert
}

func (a *promptAuditPoolFailureAlert) Observe(kind securityaudit.DecisionKind) {
	if a == nil {
		return
	}
	failure := kind == securityaudit.DecisionUnavailable || kind == securityaudit.DecisionInvalid

	a.mu.Lock()
	if !failure {
		a.consecutive = 0
		a.generation++
		a.mu.Unlock()
		return
	}
	a.consecutive++
	count, generation, notify := a.consecutive, a.generation, a.notify
	a.mu.Unlock()

	if count != promptAuditPoolFailureThreshold || notify == nil {
		return
	}
	go func() {
		if notify(count, kind) {
			return
		}
		a.mu.Lock()
		if a.generation == generation && a.consecutive >= promptAuditPoolFailureThreshold {
			a.consecutive = promptAuditPoolFailureThreshold - 1
		}
		a.mu.Unlock()
	}()
}

func (a *promptAuditPoolFailureAlert) sendEmail(count int, kind securityaudit.DecisionKind) bool {
	if a == nil || a.alerts == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), promptAuditPoolAlertTimeout)
	defer cancel()

	return a.alerts.SendPromptAuditPoolFailureEmail(ctx, count, string(kind))
}
