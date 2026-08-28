package service

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

const maxOpenAIOAuthAccessGrantUsers = 1000

type OpenAIOAuthAccessAccount struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Status             string  `json:"status"`
	GroupIDs           []int64 `json:"group_ids"`
	Mode               string  `json:"mode"`
	DefaultForNewUsers bool    `json:"default_for_new_users"`
	Revision           int64   `json:"revision"`
	GrantedUserIDs     []int64 `json:"granted_user_ids"`
}

type OpenAIOAuthAccessUser struct {
	ID                   int64   `json:"id"`
	Email                string  `json:"email"`
	Status               string  `json:"status"`
	APIKeyGroupIDs       []int64 `json:"api_key_group_ids"`
	SubscriptionGroupIDs []int64 `json:"subscription_group_ids"`
	GrantedAccountIDs    []int64 `json:"granted_account_ids"`
	EffectiveAccountIDs  []int64 `json:"effective_account_ids"`
}

type OpenAIOAuthAccessUserFilter struct {
	Search string
	Status string
	Access string
	Page   int
	Limit  int
}

type OpenAIOAuthAccessUserPage struct {
	Items []OpenAIOAuthAccessUser `json:"items"`
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	Limit int                     `json:"limit"`
	Pages int                     `json:"pages"`
}

type OpenAIOAuthAccessPolicyChange struct {
	AccountID          int64   `json:"account_id"`
	ExpectedRevision   int64   `json:"expected_revision"`
	Mode               string  `json:"mode"`
	DefaultForNewUsers bool    `json:"default_for_new_users"`
	GrantedUserIDs     []int64 `json:"granted_user_ids"`
}

type OpenAIOAuthAccessPolicyBatch struct {
	Changes []OpenAIOAuthAccessPolicyChange `json:"changes"`
}

type OpenAIOAuthAccessAccountImpact struct {
	AccountID             int64  `json:"account_id"`
	AccountName           string `json:"account_name"`
	OldMode               string `json:"old_mode"`
	NewMode               string `json:"new_mode"`
	OldDefaultForNewUsers bool   `json:"old_default_for_new_users"`
	NewDefaultForNewUsers bool   `json:"new_default_for_new_users"`
	GrantedUserCount      int    `json:"granted_user_count"`
	GrantAddedCount       int    `json:"grant_added_count"`
	GrantRemovedCount     int    `json:"grant_removed_count"`
}

type OpenAIOAuthAccessAffectedUser struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	APIKeyGroupIDs []int64 `json:"api_key_group_ids"`
}

type OpenAIOAuthAccessPreview struct {
	Accounts                  []OpenAIOAuthAccessAccountImpact `json:"accounts"`
	GrantAddedCount           int                              `json:"grant_added_count"`
	GrantRemovedCount         int                              `json:"grant_removed_count"`
	UsersLosingAllAccessCount int                              `json:"users_losing_all_access_count"`
	UsersLosingAllAccess      []OpenAIOAuthAccessAffectedUser  `json:"users_losing_all_access"`
}

type OpenAIOAuthAccessApplyResult struct {
	Accounts          []OpenAIOAuthAccessAccount `json:"accounts"`
	AccountCount      int                        `json:"account_count"`
	GrantAddedCount   int                        `json:"grant_added_count"`
	GrantRemovedCount int                        `json:"grant_removed_count"`
	AuditAccountIDs   string                     `json:"-"`
	AuditRevisions    string                     `json:"-"`
	AuditModes        string                     `json:"-"`
}

type OpenAIOAuthUserAccessRepository interface {
	ListAccounts(ctx context.Context) ([]OpenAIOAuthAccessAccount, error)
	ListUsers(ctx context.Context, search, status string) ([]OpenAIOAuthAccessUser, error)
	ApplyPolicies(ctx context.Context, changes []OpenAIOAuthAccessPolicyChange) error
}

type OpenAIOAuthUserAccessService struct {
	repo OpenAIOAuthUserAccessRepository
}

func NewOpenAIOAuthUserAccessService(repo OpenAIOAuthUserAccessRepository) *OpenAIOAuthUserAccessService {
	return &OpenAIOAuthUserAccessService{repo: repo}
}

func (s *OpenAIOAuthUserAccessService) ListAccounts(ctx context.Context) ([]OpenAIOAuthAccessAccount, error) {
	return s.repo.ListAccounts(ctx)
}

func (s *OpenAIOAuthUserAccessService) ListUsers(ctx context.Context, filter OpenAIOAuthAccessUserFilter) (*OpenAIOAuthAccessUserPage, error) {
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.ListUsers(ctx, strings.TrimSpace(filter.Search), strings.TrimSpace(filter.Status))
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].EffectiveAccountIDs = effectiveOpenAIOAuthAccountIDs(users[i], accounts)
	}
	access := strings.TrimSpace(filter.Access)
	if access != "" && access != "all" {
		filtered := users[:0]
		for _, user := range users {
			if (access == "none" && len(user.EffectiveAccountIDs) == 0) ||
				(access == "granted" && len(user.GrantedAccountIDs) > 0) {
				filtered = append(filtered, user)
			}
		}
		users = filtered
	}
	page, limit := normalizeOpenAIOAuthAccessPage(filter.Page, filter.Limit)
	total := len(users)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	pages := (total + limit - 1) / limit
	if pages < 1 {
		pages = 1
	}
	return &OpenAIOAuthAccessUserPage{
		Items: append([]OpenAIOAuthAccessUser(nil), users[start:end]...),
		Total: total,
		Page:  page,
		Limit: limit,
		Pages: pages,
	}, nil
}

func (s *OpenAIOAuthUserAccessService) Preview(ctx context.Context, batch OpenAIOAuthAccessPolicyBatch) (*OpenAIOAuthAccessPreview, error) {
	changes, err := normalizeOpenAIOAuthAccessChanges(batch.Changes)
	if err != nil {
		return nil, err
	}
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.ListUsers(ctx, "", "")
	if err != nil {
		return nil, err
	}
	accountIndex := make(map[int64]int, len(accounts))
	for i := range accounts {
		accountIndex[accounts[i].ID] = i
	}
	proposed := append([]OpenAIOAuthAccessAccount(nil), accounts...)
	preview := &OpenAIOAuthAccessPreview{}
	for _, change := range changes {
		index, ok := accountIndex[change.AccountID]
		if !ok {
			return nil, infraerrors.NotFound("OPENAI_OAUTH_ACCESS_ACCOUNT_NOT_FOUND", "OpenAI OAuth account not found")
		}
		current := accounts[index]
		if current.Revision != change.ExpectedRevision {
			return nil, OpenAIOAuthAccessRevisionConflict(change.AccountID, current.Revision)
		}
		added, removed := diffOpenAIOAuthAccessIDs(current.GrantedUserIDs, change.GrantedUserIDs)
		preview.GrantAddedCount += added
		preview.GrantRemovedCount += removed
		preview.Accounts = append(preview.Accounts, OpenAIOAuthAccessAccountImpact{
			AccountID:             current.ID,
			AccountName:           current.Name,
			OldMode:               current.Mode,
			NewMode:               change.Mode,
			OldDefaultForNewUsers: current.DefaultForNewUsers,
			NewDefaultForNewUsers: change.DefaultForNewUsers,
			GrantedUserCount:      len(change.GrantedUserIDs),
			GrantAddedCount:       added,
			GrantRemovedCount:     removed,
		})
		proposed[index].Mode = change.Mode
		proposed[index].DefaultForNewUsers = change.DefaultForNewUsers
		proposed[index].GrantedUserIDs = append([]int64(nil), change.GrantedUserIDs...)
	}
	for _, user := range users {
		if len(effectiveOpenAIOAuthAccountIDs(user, accounts)) > 0 && len(effectiveOpenAIOAuthAccountIDs(user, proposed)) == 0 {
			preview.UsersLosingAllAccess = append(preview.UsersLosingAllAccess, OpenAIOAuthAccessAffectedUser{
				ID:             user.ID,
				Email:          user.Email,
				APIKeyGroupIDs: append([]int64(nil), user.APIKeyGroupIDs...),
			})
		}
	}
	preview.UsersLosingAllAccessCount = len(preview.UsersLosingAllAccess)
	return preview, nil
}

func (s *OpenAIOAuthUserAccessService) Apply(ctx context.Context, batch OpenAIOAuthAccessPolicyBatch) (*OpenAIOAuthAccessApplyResult, error) {
	changes, err := normalizeOpenAIOAuthAccessChanges(batch.Changes)
	if err != nil {
		return nil, err
	}
	preview, err := s.Preview(ctx, OpenAIOAuthAccessPolicyBatch{Changes: changes})
	if err != nil {
		return nil, err
	}
	if err := s.repo.ApplyPolicies(ctx, changes); err != nil {
		return nil, err
	}
	result := &OpenAIOAuthAccessApplyResult{
		Accounts:          []OpenAIOAuthAccessAccount{},
		AccountCount:      len(changes),
		GrantAddedCount:   preview.GrantAddedCount,
		GrantRemovedCount: preview.GrantRemovedCount,
	}
	result.AuditAccountIDs, result.AuditRevisions, result.AuditModes = openAIOAuthAccessAuditSummary(changes, preview.Accounts)
	accounts, listErr := s.repo.ListAccounts(ctx)
	if listErr != nil {
		slog.Warn("OpenAI OAuth access policies applied but response refresh failed", "error", listErr)
		return result, nil
	}
	result.Accounts = accounts
	return result, nil
}

func normalizeOpenAIOAuthAccessChanges(changes []OpenAIOAuthAccessPolicyChange) ([]OpenAIOAuthAccessPolicyChange, error) {
	if len(changes) == 0 || len(changes) > 25 {
		return nil, infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_INVALID_CHANGES", "between 1 and 25 account changes are required")
	}
	normalized := make([]OpenAIOAuthAccessPolicyChange, 0, len(changes))
	seenAccounts := make(map[int64]struct{}, len(changes))
	for _, change := range changes {
		if change.AccountID <= 0 || change.ExpectedRevision < 0 {
			return nil, infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_INVALID_CHANGES", "account id and revision are invalid")
		}
		if _, exists := seenAccounts[change.AccountID]; exists {
			return nil, infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_DUPLICATE_ACCOUNT", "an account may appear only once")
		}
		seenAccounts[change.AccountID] = struct{}{}
		if change.Mode != OpenAIOAuthUserAccessModePublic && change.Mode != OpenAIOAuthUserAccessModeRestricted {
			return nil, infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_INVALID_MODE", "mode must be public or restricted")
		}
		change.GrantedUserIDs = uniqueSortedPositiveIDs(change.GrantedUserIDs)
		if len(change.GrantedUserIDs) > maxOpenAIOAuthAccessGrantUsers {
			return nil, infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_TOO_MANY_GRANTS", "an account may grant at most 1000 users")
		}
		if change.Mode == OpenAIOAuthUserAccessModePublic {
			if change.DefaultForNewUsers || len(change.GrantedUserIDs) > 0 {
				return nil, infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_PUBLIC_WITH_GRANTS", "public accounts cannot keep grants or a new-user default")
			}
		}
		normalized = append(normalized, change)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].AccountID < normalized[j].AccountID })
	return normalized, nil
}

func openAIOAuthAccessAuditSummary(
	changes []OpenAIOAuthAccessPolicyChange,
	impacts []OpenAIOAuthAccessAccountImpact,
) (string, string, string) {
	impactByAccount := make(map[int64]OpenAIOAuthAccessAccountImpact, len(impacts))
	for _, impact := range impacts {
		impactByAccount[impact.AccountID] = impact
	}
	accountIDs := make([]string, 0, len(changes))
	revisions := make([]string, 0, len(changes))
	modes := make([]string, 0, len(changes))
	for _, change := range changes {
		accountIDs = append(accountIDs, strconv.FormatInt(change.AccountID, 10))
		revisions = append(revisions, strconv.FormatInt(change.ExpectedRevision, 10)+">"+strconv.FormatInt(change.ExpectedRevision+1, 10))
		impact := impactByAccount[change.AccountID]
		modes = append(modes, impact.OldMode+">"+change.Mode)
	}
	return strings.Join(accountIDs, ","), strings.Join(revisions, ","), strings.Join(modes, ",")
}

func effectiveOpenAIOAuthAccountIDs(user OpenAIOAuthAccessUser, accounts []OpenAIOAuthAccessAccount) []int64 {
	groupSet := int64Set(user.APIKeyGroupIDs)
	effective := make([]int64, 0)
	for _, account := range accounts {
		if !intersectsOpenAIOAuthAccessGroups(groupSet, account.GroupIDs) {
			continue
		}
		if account.Mode == OpenAIOAuthUserAccessModePublic {
			effective = append(effective, account.ID)
			continue
		}
		if account.Mode == OpenAIOAuthUserAccessModeRestricted {
			index := sort.Search(len(account.GrantedUserIDs), func(i int) bool {
				return account.GrantedUserIDs[i] >= user.ID
			})
			if index < len(account.GrantedUserIDs) && account.GrantedUserIDs[index] == user.ID {
				effective = append(effective, account.ID)
			}
		}
	}
	return effective
}

func intersectsOpenAIOAuthAccessGroups(groupSet map[int64]struct{}, accountGroups []int64) bool {
	for _, groupID := range accountGroups {
		if _, ok := groupSet[groupID]; ok {
			return true
		}
	}
	return false
}

func int64Set(ids []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func uniqueSortedPositiveIDs(ids []int64) []int64 {
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			set[id] = struct{}{}
		}
	}
	out := make([]int64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func diffOpenAIOAuthAccessIDs(oldIDs, newIDs []int64) (added, removed int) {
	oldSet := int64Set(oldIDs)
	newSet := int64Set(newIDs)
	for id := range newSet {
		if _, ok := oldSet[id]; !ok {
			added++
		}
	}
	for id := range oldSet {
		if _, ok := newSet[id]; !ok {
			removed++
		}
	}
	return added, removed
}

func normalizeOpenAIOAuthAccessPage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

func OpenAIOAuthAccessRevisionConflict(accountID, currentRevision int64) error {
	return infraerrors.Conflict("OPENAI_OAUTH_ACCESS_REVISION_CONFLICT", "OpenAI OAuth access policy changed; refresh and try again").
		WithMetadata(map[string]string{
			"account_id":       strconv.FormatInt(accountID, 10),
			"current_revision": strconv.FormatInt(currentRevision, 10),
		})
}
