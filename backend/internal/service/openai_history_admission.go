package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
	"github.com/LuckyKuang/sub2api-plus/internal/openaiwire"
	"github.com/gin-gonic/gin"
)

type openAIHistoryContextKey struct{}

// The original owner never changes during failover. route is a separate CAS
// cursor for this request's writes and cannot authorize any candidate.
type openAIHistoryAdmission struct {
	mu              sync.Mutex
	lookupOnce      sync.Once
	service         *OpenAIGatewayService
	userID          int64
	apiKeyID        int64
	groupID         int64
	sessionHash     string
	previousID      string
	history         bool
	readOnly        bool
	oauthConsidered bool
	owner           *OpenAIConversationBinding
	route           *OpenAIConversationBinding
	err             error
	denied          map[int64]string
	responses       map[string]struct{}
	parents         map[int64]*Account
}

func openAIHistoryFromContext(ctx context.Context) *openAIHistoryAdmission {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(openAIHistoryContextKey{}).(*openAIHistoryAdmission)
	return state
}

func ContextWithOpenAIHistory(ctx, source context.Context) context.Context {
	if state := openAIHistoryFromContext(source); state != nil {
		return context.WithValue(ctx, openAIHistoryContextKey{}, state)
	}
	return ctx
}

func openAIHistoryTurnContext(ctx context.Context, hooks *OpenAIWSIngressHooks) context.Context {
	if hooks != nil && hooks.HistoryContext != nil {
		return ContextWithOpenAIHistory(ctx, hooks.HistoryContext())
	}
	return ctx
}

// PrepareOpenAIHistoryRequest must run on the authenticated, audited ingress
// body, before protocol transformations or account selection. It replaces the
// turn state for WS and leaves the original payload untouched.
func (s *OpenAIGatewayService) PrepareOpenAIHistoryRequest(c *gin.Context, protocol string, body []byte, sessionHash string, readOnly ...bool) error {
	if s == nil || c == nil || c.Request == nil {
		return ErrOpenAIHistoryUnavailable
	}
	classificationBody := body
	if protocol == ContentModerationProtocolOpenAIResponses {
		if normalized, changed := openaiwire.NormalizeCodexAutomationBootstrap(classificationBody); changed {
			classificationBody = normalized
		}
		if normalized, changed := openaiwire.NormalizeCodexDelegationBootstrap(classificationBody); changed {
			classificationBody = normalized
		}
	}
	document, err := auditcontent.Extract(protocol, classificationBody)
	if err != nil {
		return fmt.Errorf("classify conversation history: %w", err)
	}
	previousID := strings.TrimSpace(openAIRequestPayloadView(body).Get("previous_response_id").String())
	state := &openAIHistoryAdmission{
		service: s, userID: getAPIKeyUserIDFromContext(c), apiKeyID: getAPIKeyIDFromContext(c),
		groupID: getOpenAIGroupIDFromContext(c), sessionHash: sessionHash, previousID: previousID,
		history: document.HistoryBearing || document.Incomplete || previousID != "",
		denied:  make(map[int64]string), responses: make(map[string]struct{}), parents: make(map[int64]*Account),
	}
	state.readOnly = len(readOnly) > 0 && readOnly[0]
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), openAIHistoryContextKey{}, state))
	SetOpenAIHTTPResponseOwner(c, state.userID, state.apiKeyID)
	return nil
}

func (state *openAIHistoryAdmission) load(ctx context.Context) {
	state.lookupOnce.Do(func() {
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.userID <= 0 || state.apiKeyID <= 0 {
			state.err = ErrOpenAIHistoryUnavailable
			return
		}
		repo := state.service.conversationBindingRepository()
		if repo == nil {
			state.err = ErrOpenAIHistoryUnavailable
			return
		}
		lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		kind, identifier := "session", state.sessionHash
		if state.previousID != "" {
			kind, identifier = "response", state.previousID
		}
		if identifier == "" {
			return
		}
		state.owner, state.err = state.service.lookupOpenAIHistoryBinding(lookupCtx, state.userID, state.groupID, kind, identifier)
		if state.err == nil && state.owner == nil && kind == "response" {
			state.owner, state.err = state.service.restoreOpenAIHistoryResponseBinding(lookupCtx, state, identifier)
		}
		if state.err != nil {
			slog.WarnContext(ctx, "history_binding_lookup_failed", "error", state.err)
			return
		}
		if state.owner != nil {
			state.history = true
		}
		if kind == "session" {
			state.route = state.owner
		}
	})
}

func (s *OpenAIGatewayService) lookupOpenAIHistoryBinding(ctx context.Context, userID, groupID int64, kind, identifier string) (*OpenAIConversationBinding, error) {
	repo := s.conversationBindingRepository()
	if repo == nil {
		return nil, ErrOpenAIHistoryUnavailable
	}
	key := openAIConversationBindingKey(identifier)
	binding, err := repo.GetOpenAIConversationBinding(ctx, userID, groupID, kind, key)
	if err != nil || binding != nil {
		return binding, err
	}
	binding, err = repo.GetOpenAIConversationBinding(ctx, userID, openAIOAuthSharedSessionCacheGroupID, kind, key)
	if err != nil || binding == nil || !binding.Valid {
		return binding, err
	}
	account, err := s.accountRepo.GetByID(ctx, binding.AccountID)
	if err != nil {
		return nil, err
	}
	if account == nil || !account.IsOpenAIOAuthSessionSharingEnabled() ||
		!openAIAccountAllowsEffectiveGroup(account, &groupID, false) || openAIOAuthUserAccessFailureReason(ctx, account) != "" {
		return nil, nil
	}
	policy, _, valid := account.OpenAIOAuthSessionPolicy()
	if !valid || policy.ScopeVersion != binding.PolicyScopeVersion {
		binding.Valid = false
	}
	return binding, nil
}

func openAIHistoryCandidateFailureReason(ctx context.Context, account *Account) string {
	state := openAIHistoryFromContext(ctx)
	if state == nil || account == nil || !account.IsOpenAIOAuth() {
		return ""
	}
	state.mu.Lock()
	state.oauthConsidered = true
	state.mu.Unlock()
	state.load(ctx)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.err != nil {
		return "history_binding_lookup_failed"
	}
	policyAccount := account
	ownerID := account.ID
	if account.IsCredentialShadow() && account.ParentAccountID != nil {
		ownerID = *account.ParentAccountID
		parent := state.parents[ownerID]
		if parent == nil {
			var err error
			parent, err = state.service.accountRepo.GetByID(ctx, ownerID)
			if err != nil || parent == nil || !parent.IsOpenAIOAuth() {
				state.err = ErrOpenAIHistoryUnavailable
				return "history_binding_lookup_failed"
			}
			state.parents[ownerID] = parent
		}
		policyAccount = parent
	}
	if !state.history || !policyAccount.IsOpenAIOAuthRejectExternalHistoryEnabled() {
		return ""
	}
	if state.owner != nil && state.owner.Valid && state.owner.CredentialOwnerAccountID == ownerID {
		if openAIHistoryBindingMatchesIdentity(state.owner, policyAccount) {
			return ""
		}
		// Lightweight/stale scheduling rows cannot establish credential identity.
		// Re-read the owner before deciding, including a reauthorization between
		// the initial lookup and a retry or concurrency wait.
		latest, err := state.service.accountRepo.GetByID(ctx, ownerID)
		if err != nil || latest == nil {
			state.err = ErrOpenAIHistoryUnavailable
			return "history_binding_lookup_failed"
		}
		if !latest.IsOpenAIOAuthRejectExternalHistoryEnabled() || openAIHistoryBindingMatchesIdentity(state.owner, latest) {
			return ""
		}
	}
	reason := "history_owner_missing"
	if state.owner != nil && state.owner.Valid {
		reason = "history_owner_mismatch"
	}
	if _, exists := state.denied[account.ID]; !exists {
		slog.InfoContext(ctx, reason, "account_id", account.ID)
		state.denied[account.ID] = reason
	}
	return reason
}

func openAIHistoryBindingMatchesIdentity(binding *OpenAIConversationBinding, account *Account) bool {
	if binding == nil || account == nil || !account.IsOpenAIOAuth() {
		return false
	}
	policy, _, valid := account.OpenAIOAuthSessionPolicy()
	return valid && binding.OAuthAccountID == account.GetCredential("chatgpt_account_id") &&
		binding.OAuthUserID == account.GetCredential("chatgpt_user_id") && binding.PolicyScopeVersion == policy.ScopeVersion
}

func openAIHistorySelectionError(ctx context.Context, err error) error {
	state := openAIHistoryFromContext(ctx)
	if state == nil || !isOpenAIAccountSelectionUnavailable(err) {
		return err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.oauthConsidered {
		return err
	}
	if state.err != nil {
		return fmt.Errorf("%w: %w", ErrOpenAIHistoryUnavailable, state.err)
	}
	if len(state.denied) > 0 {
		slog.InfoContext(ctx, "history_policy_candidates_exhausted", "filtered_accounts", len(state.denied))
		return ErrOpenAIExternalHistory
	}
	return err
}

func openAIHistoryStickyAccountID(ctx context.Context, sessionHash string) int64 {
	state := openAIHistoryFromContext(ctx)
	if state == nil {
		return 0
	}
	state.load(ctx)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.owner != nil && state.owner.Valid && state.err == nil &&
		(state.previousID != "" || state.sessionHash == sessionHash) {
		return state.owner.AccountID
	}
	return 0
}

func (s *OpenAIGatewayService) newOpenAIHistoryBinding(ctx context.Context, state *openAIHistoryAdmission, account *Account, kind, identifier string) (*OpenAIConversationBinding, error) {
	owner := account
	if account.IsCredentialShadow() && account.ParentAccountID != nil {
		var err error
		owner, err = s.accountRepo.GetByID(ctx, *account.ParentAccountID)
		if err != nil || owner == nil || !owner.IsOpenAIOAuth() {
			return nil, ErrOpenAIHistoryUnavailable
		}
	}
	groupID := state.groupID
	policy, _, valid := owner.OpenAIOAuthSessionPolicy()
	if !valid {
		return nil, ErrOpenAIOAuthSessionAccessDenied
	}
	if policy.Enabled {
		if !openAIAccountAllowsEffectiveGroup(account, &groupID, false) {
			return nil, ErrOpenAIOAuthSessionAccessDenied
		}
		groupID = openAIOAuthSharedSessionCacheGroupID
	}
	return &OpenAIConversationBinding{
		UserID: state.userID, APIKeyID: state.apiKeyID, ScopeGroupID: groupID,
		Type: kind, Key: openAIConversationBindingKey(identifier), AccountID: account.ID,
		CredentialOwnerAccountID: owner.ID, OAuthAccountID: owner.GetCredential("chatgpt_account_id"),
		OAuthUserID: owner.GetCredential("chatgpt_user_id"), PolicyScopeVersion: policy.ScopeVersion,
	}, nil
}

// CommitOpenAIHistoryRoute runs after candidate admission, before forwarding.
// It is independent from best-effort Redis sticky writes inside the scheduler.
func (s *OpenAIGatewayService) CommitOpenAIHistoryRoute(ctx context.Context, account *Account) error {
	state := openAIHistoryFromContext(ctx)
	if state == nil || account == nil || !account.IsOpenAIOAuth() {
		return nil
	}
	if reason := openAIHistoryCandidateFailureReason(ctx, account); reason != "" {
		if reason == "history_binding_lookup_failed" {
			return ErrOpenAIHistoryUnavailable
		}
		return ErrOpenAIExternalHistory
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.sessionHash == "" || state.readOnly {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	binding, err := s.newOpenAIHistoryBinding(writeCtx, state, account, "session", state.sessionHash)
	if err != nil {
		return err
	}
	repo := s.conversationBindingRepository()
	if repo == nil {
		return ErrOpenAIHistoryUnavailable
	}
	if state.route == nil && state.previousID != "" {
		state.route, err = repo.GetOpenAIConversationBinding(writeCtx, state.userID, binding.ScopeGroupID, binding.Type, binding.Key)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrOpenAIHistoryUnavailable, err)
		}
	}
	expectedRevision := int64(0)
	if state.route != nil && state.route.ScopeGroupID == binding.ScopeGroupID {
		expectedRevision = state.route.Revision
	}
	saved, err := repo.SaveOpenAIConversationBinding(writeCtx, binding, expectedRevision)
	if err != nil {
		if errors.Is(err, ErrOpenAIHistoryConflict) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrOpenAIHistoryUnavailable, err)
	}
	state.route = saved
	return nil
}

func (s *OpenAIGatewayService) persistOpenAIHistoryResponse(ctx context.Context, account *Account, responseID string) error {
	state := openAIHistoryFromContext(ctx)
	responseID = strings.TrimSpace(responseID)
	if state == nil || state.readOnly || account == nil || !account.IsOpenAIOAuth() || responseID == "" {
		return nil
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	key := fmt.Sprintf("%d:%s", account.ID, responseID)
	if _, exists := state.responses[key]; exists {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	binding, err := s.newOpenAIHistoryBinding(writeCtx, state, account, "response", responseID)
	if err != nil {
		return err
	}
	repo := s.conversationBindingRepository()
	if repo == nil {
		return ErrOpenAIHistoryUnavailable
	}
	if _, err := repo.SaveOpenAIConversationBinding(writeCtx, binding, 0); err != nil {
		slog.WarnContext(ctx, "history_binding_write_failed", "account_id", account.ID, "error", err)
		return fmt.Errorf("%w: %w", ErrOpenAIHistoryUnavailable, err)
	}
	state.responses[key] = struct{}{}
	return nil
}

func (s *OpenAIGatewayService) persistOpenAIHistoryResponsePayload(ctx context.Context, account *Account, payload []byte) error {
	if openAIHistoryFromContext(ctx) == nil || account == nil || !account.IsOpenAIOAuth() {
		return nil
	}
	if !bodyHasSSEFraming(payload) {
		return s.persistOpenAIHistoryResponse(ctx, account, extractOpenAIResponseIDFromJSONBytes(payload))
	}
	var bindErr error
	forEachOpenAISSEFrame(string(payload), func(_ string, data []byte) {
		if bindErr == nil {
			bindErr = s.persistOpenAIHistoryResponse(ctx, account, extractOpenAIResponseIDFromJSONBytes(data))
		}
	})
	return bindErr
}

func (s *OpenAIGatewayService) restoreOpenAIHistoryResponseBinding(ctx context.Context, state *openAIHistoryAdmission, responseID string) (*OpenAIConversationBinding, error) {
	store := s.getOpenAIWSStateStore()
	userID, _, found, err := store.GetHTTPResponseOwner(ctx, state.groupID, responseID)
	if errors.Is(err, ErrGatewayCacheMiss) {
		return nil, nil
	}
	if err != nil || !found || userID != state.userID {
		return nil, err
	}
	accountID, err := store.GetResponseAccount(ctx, state.groupID, responseID)
	if errors.Is(err, ErrGatewayCacheMiss) {
		return nil, nil
	}
	if err != nil || accountID <= 0 {
		return nil, err
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil || !account.IsOpenAIOAuth() || !s.openAIAccountMatchesSchedulingGroup(account, &state.groupID) ||
		openAIOAuthUserAccessFailureReason(ctx, account) != "" {
		return nil, nil
	}
	if err := s.validateOpenAISharedPreviousResponseAccountSelection(ctx, &state.groupID, responseID, account); err != nil {
		return nil, err
	}
	binding, err := s.newOpenAIHistoryBinding(ctx, state, account, "response", responseID)
	if err != nil {
		return nil, err
	}
	if state.readOnly {
		binding.Valid = true
		return binding, nil
	}
	return s.conversationBindingRepository().SaveOpenAIConversationBinding(ctx, binding, 0)
}

// ValidateOpenAIHistoryTurn checks current persisted configuration even in WS
// passthrough mode, which does not invoke the ordinary BeforeTurn hook.
func (s *OpenAIGatewayService) ValidateOpenAIHistoryTurn(ctx context.Context, account *Account) error {
	state := openAIHistoryFromContext(ctx)
	if state == nil || account == nil || !account.IsOpenAIOAuth() {
		return nil
	}
	latest, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil || latest == nil {
		return ErrOpenAIHistoryUnavailable
	}
	if latest.IsCredentialShadow() && latest.ParentAccountID != nil {
		state.mu.Lock()
		delete(state.parents, *latest.ParentAccountID)
		state.mu.Unlock()
	}
	return s.CommitOpenAIHistoryRoute(ctx, latest)
}
