package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/stretchr/testify/require"
)

type promptAuditAlertCall struct {
	count int
	kind  securityaudit.DecisionKind
}

func TestPromptAuditPoolFailureAlertThresholdAndReset(t *testing.T) {
	calls := make(chan promptAuditAlertCall, 4)
	alert := &promptAuditPoolFailureAlert{notify: func(count int, kind securityaudit.DecisionKind) bool {
		calls <- promptAuditAlertCall{count: count, kind: kind}
		return true
	}}
	metrics := securityaudit.NewAtomicMetrics()
	metrics.SetOutcomeObserver(alert.Observe)

	for range 4 {
		metrics.ObservePoolOutcome(securityaudit.DecisionUnavailable)
	}
	select {
	case call := <-calls:
		t.Fatalf("unexpected alert before threshold: %+v", call)
	default:
	}

	metrics.ObservePoolOutcome(securityaudit.DecisionInvalid)
	require.Equal(t, promptAuditAlertCall{count: 5, kind: securityaudit.DecisionInvalid}, receivePromptAuditAlert(t, calls))
	metrics.ObservePoolOutcome(securityaudit.DecisionUnavailable)
	select {
	case call := <-calls:
		t.Fatalf("unexpected duplicate alert: %+v", call)
	case <-time.After(20 * time.Millisecond):
	}

	metrics.ObservePoolOutcome(securityaudit.DecisionBlock)
	for range 5 {
		metrics.ObservePoolOutcome(securityaudit.DecisionUnavailable)
	}
	require.Equal(t, promptAuditAlertCall{count: 5, kind: securityaudit.DecisionUnavailable}, receivePromptAuditAlert(t, calls))
}

func TestPromptAuditPoolFailureAlertConcurrentFailuresSendOnce(t *testing.T) {
	var sent atomic.Int64
	alert := &promptAuditPoolFailureAlert{notify: func(int, securityaudit.DecisionKind) bool {
		sent.Add(1)
		return true
	}}

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			alert.Observe(securityaudit.DecisionUnavailable)
		}()
	}
	wg.Wait()
	require.Eventually(t, func() bool { return sent.Load() == 1 }, time.Second, time.Millisecond)
	require.Never(t, func() bool { return sent.Load() > 1 }, 20*time.Millisecond, time.Millisecond)
}

func TestPromptAuditPoolFailureAlertRetriesFailedDelivery(t *testing.T) {
	attempts := make(chan int, 2)
	var attemptCount atomic.Int64
	alert := &promptAuditPoolFailureAlert{notify: func(count int, _ securityaudit.DecisionKind) bool {
		attempts <- count
		return attemptCount.Add(1) > 1
	}}
	for range 5 {
		alert.Observe(securityaudit.DecisionUnavailable)
	}
	require.Equal(t, 5, receivePromptAuditAlertAttempt(t, attempts))
	require.Eventually(t, func() bool {
		alert.mu.Lock()
		defer alert.mu.Unlock()
		return alert.consecutive == promptAuditPoolFailureThreshold-1
	}, time.Second, time.Millisecond)

	alert.Observe(securityaudit.DecisionUnavailable)
	require.Equal(t, 5, receivePromptAuditAlertAttempt(t, attempts))
}

func receivePromptAuditAlert(t *testing.T, calls <-chan promptAuditAlertCall) promptAuditAlertCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt audit alert")
		return promptAuditAlertCall{}
	}
}

func receivePromptAuditAlertAttempt(t *testing.T, attempts <-chan int) int {
	t.Helper()
	select {
	case count := <-attempts:
		return count
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt audit alert attempt")
		return 0
	}
}
