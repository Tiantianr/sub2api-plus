package securityaudit

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type LegacyEngine interface {
	Check(ctx context.Context, req Request) (*LegacyDecision, error)
}

type PromptEngine interface {
	EffectiveMode() Mode
	Enqueue(ctx context.Context, req Request) error
	Evaluate(ctx context.Context, req Request) (*PromptDecision, error)
}

type blockingScopePromptEngine interface {
	BlockingApplies(req Request) bool
}

type deepReviewPromptEngine interface {
	EnqueueDeep(ctx context.Context, req Request) error
}

type allowReceiptCommitter interface {
	commitAllowReceipts(ctx context.Context, decision *PromptDecision)
}

type recoveryFencePromptEngine interface {
	pendingRecoveryDecision(ctx context.Context, req Request) (*PromptDecision, error)
}

type Coordinator struct {
	legacy LegacyEngine
	prompt PromptEngine
}

func NewCoordinator(legacy LegacyEngine, prompt PromptEngine) *Coordinator {
	return &Coordinator{legacy: legacy, prompt: prompt}
}

func (c *Coordinator) Check(ctx context.Context, req Request) Decision {
	if c == nil {
		return allowDecision(nil, nil)
	}
	mode := ModeOff
	if c.prompt != nil {
		mode = c.prompt.EffectiveMode()
	}
	switch mode {
	case ModeAsync:
		legacy, _ := c.checkLegacy(ctx, req)
		decision := prioritize(legacy, nil)
		enqueueReq := req
		enqueueReq.AllowReceiptWrite = decision.AllowNextStage
		// Enqueue remains best-effort and still records blocked requests. Only a
		// combined Allow lets the eventual async job create reusable receipts.
		_ = c.prompt.Enqueue(ctx, enqueueReq)
		return decision
	case ModeBlocking:
		return c.checkBlocking(ctx, req)
	default:
		legacy, _ := c.checkLegacy(ctx, req)
		return prioritize(legacy, nil)
	}
}

func (c *Coordinator) checkBlocking(ctx context.Context, req Request) Decision {
	// Both synchronous engines treat the frozen request body as immutable.
	// EnqueueDeep takes the only copy needed beyond this request lifetime.
	legacyReq := req
	if scoped, ok := c.prompt.(blockingScopePromptEngine); ok {
		legacyReq.PromptTextAuthority = scoped.BlockingApplies(req)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	var legacy *LegacyDecision
	var prompt *PromptDecision
	go func() {
		defer wg.Done()
		legacy, _ = c.checkLegacy(ctx, legacyReq)
	}()
	go func() {
		defer wg.Done()
		if c.prompt == nil {
			prompt = unavailablePromptDecision(ErrorCodeUnavailable)
			return
		}
		result, err := c.prompt.Evaluate(ctx, req)
		if err != nil {
			var guardErr *GuardError
			if errors.As(err, &guardErr) && strings.TrimSpace(guardErr.Code) != "" {
				prompt = unavailablePromptDecision(guardErr.Code)
				return
			}
			prompt = unavailablePromptDecision(ErrorCodeUnavailable)
			return
		}
		if result == nil {
			prompt = unavailablePromptDecision(ErrorCodeUnavailable)
			return
		}
		prompt = result
	}()
	wg.Wait()
	decision := prioritize(legacy, prompt)
	if decision.AllowNextStage {
		if fence, ok := c.prompt.(recoveryFencePromptEngine); ok {
			pending, err := fence.pendingRecoveryDecision(ctx, req)
			if err != nil {
				decision = prioritize(legacy, unavailablePromptDecision(ErrorCodeDeepReviewState))
			} else if pending != nil {
				decision = prioritize(legacy, pending)
			}
		}
	}
	if decision.AllowNextStage {
		if committer, ok := c.prompt.(allowReceiptCommitter); ok {
			committer.commitAllowReceipts(ctx, prompt)
		}
	}
	if decision.AllowNextStage && prompt != nil && !prompt.DeepReviewed {
		if deep, ok := c.prompt.(deepReviewPromptEngine); ok {
			deepReq := req
			deepReq.AllowReceiptKeys = append([]string(nil), prompt.AllowReceiptKeys...)
			deepReq.AllowReceiptWrite = !prompt.FailureAllowed
			deepReq.SuppressReceiptWrite = prompt.FailureAllowed
			_ = deep.EnqueueDeep(ctx, deepReq)
		}
	}
	return decision
}

func (c *Coordinator) checkLegacy(ctx context.Context, req Request) (*LegacyDecision, error) {
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.Check(ctx, req)
}

func prioritize(legacy *LegacyDecision, prompt *PromptDecision) Decision {
	if legacy != nil && legacy.Blocked && !legacyDecisionUnavailable(legacy) {
		status := legacy.StatusCode
		if status < 400 || status > 599 {
			status = http.StatusForbidden
		}
		code := legacy.ErrorCode
		if code == "" {
			code = "content_policy_violation"
		}
		return Decision{
			Kind: DecisionBlock, HTTPStatus: status, ErrorCode: code, ClientMessage: legacy.Message,
			Legacy: legacy, Prompt: prompt, AllowNextStage: false,
		}
	}
	if prompt != nil && prompt.Kind == DecisionBlock {
		code := prompt.ErrorCode
		if code == "" {
			code = ErrorCodeBlocked
		}
		return Decision{Kind: DecisionBlock, HTTPStatus: http.StatusForbidden, ErrorCode: code,
			ClientMessage: promptBlockClientMessage(prompt), Legacy: legacy, Prompt: prompt}
	}
	if legacyDecisionUnavailable(legacy) {
		status := legacy.StatusCode
		if status < 500 || status > 599 {
			status = http.StatusServiceUnavailable
		}
		code := legacy.ErrorCode
		if code == "" {
			code = service.ContentModerationErrorCodeUnavailable
		}
		return Decision{
			Kind: DecisionUnavailable, HTTPStatus: status, ErrorCode: code, ClientMessage: legacy.Message,
			Legacy: legacy, Prompt: prompt, AllowNextStage: false,
		}
	}
	if prompt == nil {
		return allowDecision(legacy, nil)
	}
	switch prompt.Kind {
	case DecisionInvalid:
		code := prompt.ErrorCode
		if code == "" {
			code = ErrorCodeInvalidResponse
		}
		return Decision{Kind: DecisionInvalid, HTTPStatus: http.StatusServiceUnavailable, ErrorCode: code,
			ClientMessage: "提示词安全审计暂时不可用，请稍后重试", Legacy: legacy, Prompt: prompt}
	case DecisionUnavailable:
		code := prompt.ErrorCode
		if code == "" {
			code = ErrorCodeUnavailable
		}
		return Decision{Kind: DecisionUnavailable, HTTPStatus: http.StatusServiceUnavailable, ErrorCode: code,
			ClientMessage: promptUnavailableClientMessage(code), Legacy: legacy, Prompt: prompt}
	case DecisionFlag:
		return Decision{Kind: DecisionFlag, HTTPStatus: http.StatusOK, Legacy: legacy, Prompt: prompt, AllowNextStage: true}
	default:
		return allowDecision(legacy, prompt)
	}
}

func promptUnavailableClientMessage(code string) string {
	if code == ErrorCodeDeepReviewState {
		return "安全审计恢复状态暂时不可用，请稍后重试"
	}
	return "提示词安全审计暂时不可用，请稍后重试"
}

func promptBlockClientMessage(prompt *PromptDecision) string {
	const fallback = "安全审计拒绝了本次请求。风险内容可能来自当前输入、会话上下文、系统指令或插件/工具内容，请检查并移除相关内容后重试"
	if prompt == nil || prompt.Result == nil {
		return fallback
	}
	reasons := make([]string, 0, len(prompt.Result.Categories)+1)
	seen := make(map[string]struct{}, len(prompt.Result.Categories)+1)
	for _, summary := range BuildIssueSummaries(*prompt.Result) {
		title := strings.TrimSpace(summary.Title)
		if title == "" {
			continue
		}
		if _, exists := seen[title]; exists {
			continue
		}
		seen[title] = struct{}{}
		reasons = append(reasons, title)
	}
	if len(reasons) == 0 {
		return fallback
	}
	return "安全审计拒绝了本次请求，命中风险类别：" + strings.Join(reasons, "、") + "。风险内容可能来自当前输入、会话上下文、系统指令或插件/工具内容，请检查并移除相关内容后重试"
}

func legacyDecisionUnavailable(decision *LegacyDecision) bool {
	if decision == nil {
		return false
	}
	if decision.ErrorCode == service.ContentModerationErrorCodeUnavailable {
		return true
	}
	return decision.Action == service.ContentModerationActionError && !decision.Flagged
}

func allowDecision(legacy *LegacyDecision, prompt *PromptDecision) Decision {
	return Decision{Kind: DecisionAllow, HTTPStatus: http.StatusOK, Legacy: legacy, Prompt: prompt, AllowNextStage: true}
}

func unavailablePromptDecision(code string) *PromptDecision {
	kind := DecisionUnavailable
	if code == ErrorCodeInvalidResponse {
		kind = DecisionInvalid
	}
	return &PromptDecision{Kind: kind, ErrorCode: code, AllowNextStage: false}
}
