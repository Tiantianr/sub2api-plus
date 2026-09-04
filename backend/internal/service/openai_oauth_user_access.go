package service

import (
	"context"
	"errors"
	"sort"
)

const (
	OpenAIOAuthUserAccessModePublic     = "public"
	OpenAIOAuthUserAccessModeRestricted = "restricted"
	openAIOAuthUserAccessDeniedReason   = "oauth_user_access_denied"
	openAIOAuthUserAccessRecheckReason  = "oauth_user_access_recheck_failed"
)

var ErrOpenAIOAuthUserAccessDenied = errors.New("OpenAI account is not available for this user")

// OpenAIOAuthUserAccessSnapshot is the scheduler-safe projection of one OpenAI
// root account's local-user access policy. A nil snapshot preserves public access.
type OpenAIOAuthUserAccessSnapshot struct {
	Mode               string  `json:"mode"`
	DefaultForNewUsers bool    `json:"default_for_new_users,omitempty"`
	Revision           int64   `json:"revision"`
	GrantedUserIDs     []int64 `json:"granted_user_ids,omitempty"`
}

func (s *OpenAIOAuthUserAccessSnapshot) Clone() *OpenAIOAuthUserAccessSnapshot {
	if s == nil {
		return nil
	}
	clone := *s
	clone.GrantedUserIDs = append([]int64(nil), s.GrantedUserIDs...)
	return &clone
}

func (s *OpenAIOAuthUserAccessSnapshot) AllowsUser(userID int64) bool {
	if s == nil || s.Mode == OpenAIOAuthUserAccessModePublic {
		return true
	}
	if s.Mode != OpenAIOAuthUserAccessModeRestricted || userID <= 0 {
		return false
	}
	index := sort.Search(len(s.GrantedUserIDs), func(i int) bool {
		return s.GrantedUserIDs[i] >= userID
	})
	return index < len(s.GrantedUserIDs) && s.GrantedUserIDs[index] == userID
}

func openAIOAuthUserAccessFailureReason(ctx context.Context, account *Account) string {
	if account == nil || account.Platform != PlatformOpenAI ||
		(account.Type != AccountTypeOAuth && account.Type != AccountTypeAPIKey) {
		return ""
	}
	if account.OpenAIOAuthUserAccess.AllowsUser(openAIRequestUserID(ctx)) {
		return ""
	}
	return openAIOAuthUserAccessDeniedReason
}

// EffectiveOpenAIAccountGroupIDs returns the persisted group bindings that are
// also authorized by an enabled OpenAI OAuth session-sharing policy.
func EffectiveOpenAIAccountGroupIDs(account *Account) []int64 {
	bound := accountBoundGroupIDs(account)
	if account == nil || !account.IsOpenAIOAuthSessionSharingEnabled() {
		return bound
	}
	policy, _, valid := account.OpenAIOAuthSessionPolicy()
	if !valid || !policy.Enabled {
		return nil
	}
	allowed := int64Set(policy.AllowedGroupIDs)
	effective := make([]int64, 0, len(bound))
	for _, groupID := range bound {
		if _, ok := allowed[groupID]; ok {
			effective = append(effective, groupID)
		}
	}
	return effective
}

func openAIAccountAllowsEffectiveGroup(account *Account, groupID *int64, simpleMode bool) bool {
	if account == nil {
		return false
	}
	if account.IsOpenAIOAuthSessionSharingEnabled() && !account.IsOpenAIOAuthSessionGroupAllowed(groupID) {
		return false
	}
	if simpleMode {
		return true
	}
	return openAIStickyAccountMatchesGroup(account, groupID)
}
