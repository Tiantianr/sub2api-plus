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

var ErrOpenAIOAuthUserAccessDenied = errors.New("OpenAI OAuth account is not available for this user")

// OpenAIOAuthUserAccessSnapshot is the scheduler-safe projection of one root
// account's local-user access policy. A nil snapshot preserves public access.
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
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeOAuth {
		return ""
	}
	if account.OpenAIOAuthUserAccess.AllowsUser(openAIRequestUserID(ctx)) {
		return ""
	}
	return openAIOAuthUserAccessDeniedReason
}
