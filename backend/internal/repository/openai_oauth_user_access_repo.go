package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/lib/pq"
)

type openAIOAuthUserAccessRepository struct {
	db             *sql.DB
	accountRepo    service.AccountRepository
	schedulerCache service.SchedulerCache
}

func NewOpenAIOAuthUserAccessRepository(
	db *sql.DB,
	accountRepo service.AccountRepository,
	schedulerCache service.SchedulerCache,
) service.OpenAIOAuthUserAccessRepository {
	return &openAIOAuthUserAccessRepository{
		db:             db,
		accountRepo:    accountRepo,
		schedulerCache: schedulerCache,
	}
}

func (r *openAIOAuthUserAccessRepository) ListAccounts(ctx context.Context) ([]service.OpenAIOAuthAccessAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			a.id,
			a.name,
			a.type,
			a.status,
			COALESCE(a.extra, '{}'::jsonb),
			COALESCE(groups.group_ids, '{}'::bigint[]),
			COALESCE(p.mode, 'public'),
			COALESCE(p.default_for_new_users, FALSE),
			COALESCE(p.revision, 0),
			COALESCE(grants.user_ids, '{}'::bigint[])
		FROM accounts AS a
		LEFT JOIN openai_oauth_account_access_policies AS p ON p.account_id = a.id
		LEFT JOIN LATERAL (
			SELECT array_agg(DISTINCT ag.group_id ORDER BY ag.group_id) AS group_ids
			FROM account_groups AS ag
			JOIN accounts AS candidate ON candidate.id = ag.account_id
			WHERE (candidate.id = a.id OR candidate.parent_account_id = a.id)
			  AND candidate.status = 'active'
			  AND candidate.schedulable = TRUE
			  AND candidate.deleted_at IS NULL
			  AND (candidate.expires_at IS NULL OR candidate.expires_at > NOW() OR candidate.auto_pause_on_expired = FALSE)
			  AND (candidate.temp_unschedulable_until IS NULL OR candidate.temp_unschedulable_until <= NOW())
			  AND (candidate.overload_until IS NULL OR candidate.overload_until <= NOW())
			  AND (candidate.rate_limit_reset_at IS NULL OR candidate.rate_limit_reset_at <= NOW())
			  AND a.status = 'active'
			  AND (a.expires_at IS NULL OR a.expires_at > NOW() OR a.auto_pause_on_expired = FALSE)
			  AND (a.temp_unschedulable_until IS NULL OR a.temp_unschedulable_until <= NOW())
		) AS groups ON TRUE
		LEFT JOIN LATERAL (
			SELECT array_agg(g.user_id ORDER BY g.user_id) AS user_ids
			FROM openai_oauth_account_user_grants AS g
			JOIN users AS u ON u.id = g.user_id AND u.deleted_at IS NULL
			WHERE g.account_id = a.id
		) AS grants ON TRUE
		WHERE a.platform = 'openai'
		  AND a.type IN ('oauth', 'apikey')
		  AND a.parent_account_id IS NULL
		  AND a.deleted_at IS NULL
		ORDER BY a.priority, a.id
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	accounts := make([]service.OpenAIOAuthAccessAccount, 0)
	for rows.Next() {
		var account service.OpenAIOAuthAccessAccount
		var groupIDs, userIDs pq.Int64Array
		var extraJSON []byte
		if err := rows.Scan(
			&account.ID,
			&account.Name,
			&account.Type,
			&account.Status,
			&extraJSON,
			&groupIDs,
			&account.Mode,
			&account.DefaultForNewUsers,
			&account.Revision,
			&userIDs,
		); err != nil {
			return nil, err
		}
		var extra map[string]any
		if err := json.Unmarshal(extraJSON, &extra); err != nil {
			return nil, fmt.Errorf("decode OpenAI account %d extra: %w", account.ID, err)
		}
		account.GroupIDs = service.EffectiveOpenAIAccountGroupIDs(&service.Account{
			ID: account.ID, Platform: service.PlatformOpenAI, Type: account.Type,
			Extra: extra, GroupIDs: []int64(groupIDs),
		})
		account.GrantedUserIDs = []int64(userIDs)
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (r *openAIOAuthUserAccessRepository) ListUsers(ctx context.Context, search, status string) ([]service.OpenAIOAuthAccessUser, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id,
			u.email,
			u.status,
			COALESCE(keys.group_ids, '{}'::bigint[]),
			COALESCE(subscriptions.group_ids, '{}'::bigint[]),
			COALESCE(grants.account_ids, '{}'::bigint[])
		FROM users AS u
		LEFT JOIN LATERAL (
			SELECT array_agg(DISTINCT k.group_id ORDER BY k.group_id) AS group_ids
			FROM api_keys AS k
			WHERE k.user_id = u.id
			  AND k.group_id IS NOT NULL
			  AND k.status = 'active'
			  AND k.deleted_at IS NULL
			  AND (k.expires_at IS NULL OR k.expires_at > NOW())
		) AS keys ON TRUE
		LEFT JOIN LATERAL (
			SELECT array_agg(DISTINCT s.group_id ORDER BY s.group_id) AS group_ids
			FROM user_subscriptions AS s
			WHERE s.user_id = u.id
			  AND s.status = 'active'
			  AND s.deleted_at IS NULL
			  AND s.starts_at <= NOW()
			  AND s.expires_at > NOW()
		) AS subscriptions ON TRUE
		LEFT JOIN LATERAL (
			SELECT array_agg(g.account_id ORDER BY g.account_id) AS account_ids
			FROM openai_oauth_account_user_grants AS g
			JOIN openai_oauth_account_access_policies AS p
			  ON p.account_id = g.account_id AND p.mode = 'restricted'
			WHERE g.user_id = u.id
		) AS grants ON TRUE
		WHERE u.role IN ('user', 'admin')
		  AND u.deleted_at IS NULL
		  AND ($1 = '' OR u.email ILIKE '%' || $1 || '%' OR u.id::text = $1)
		  AND ($2 = '' OR u.status = $2)
		ORDER BY u.id DESC
	`, strings.TrimSpace(search), strings.TrimSpace(status))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := make([]service.OpenAIOAuthAccessUser, 0)
	for rows.Next() {
		var user service.OpenAIOAuthAccessUser
		var keyGroups, subscriptionGroups, accountIDs pq.Int64Array
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Status,
			&keyGroups,
			&subscriptionGroups,
			&accountIDs,
		); err != nil {
			return nil, err
		}
		user.APIKeyGroupIDs = []int64(keyGroups)
		user.SubscriptionGroupIDs = []int64(subscriptionGroups)
		user.GrantedAccountIDs = []int64(accountIDs)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *openAIOAuthUserAccessRepository) ApplyPolicies(ctx context.Context, changes []service.OpenAIOAuthAccessPolicyChange) error {
	if len(changes) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	accountIDs := make([]int64, 0, len(changes))
	for _, change := range changes {
		accountIDs = append(accountIDs, change.AccountID)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM accounts
		WHERE id = ANY($1)
		  AND platform = 'openai'
		  AND type IN ('oauth', 'apikey')
		  AND parent_account_id IS NULL
		  AND deleted_at IS NULL
		ORDER BY id
		FOR UPDATE
	`, pq.Array(accountIDs))
	if err != nil {
		return err
	}
	lockedAccounts := make(map[int64]struct{}, len(accountIDs))
	for rows.Next() {
		var accountID int64
		if err := rows.Scan(&accountID); err != nil {
			_ = rows.Close()
			return err
		}
		lockedAccounts[accountID] = struct{}{}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(lockedAccounts) != len(accountIDs) {
		return infraerrors.NotFound("OPENAI_OAUTH_ACCESS_ACCOUNT_NOT_FOUND", "OpenAI account not found")
	}

	policyRows, err := tx.QueryContext(ctx, `
		SELECT account_id, revision
		FROM openai_oauth_account_access_policies
		WHERE account_id = ANY($1)
		ORDER BY account_id
		FOR UPDATE
	`, pq.Array(accountIDs))
	if err != nil {
		return err
	}
	revisions := make(map[int64]int64, len(accountIDs))
	for policyRows.Next() {
		var accountID, revision int64
		if err := policyRows.Scan(&accountID, &revision); err != nil {
			_ = policyRows.Close()
			return err
		}
		revisions[accountID] = revision
	}
	if err := policyRows.Close(); err != nil {
		return err
	}
	for _, change := range changes {
		if revisions[change.AccountID] != change.ExpectedRevision {
			return service.OpenAIOAuthAccessRevisionConflict(change.AccountID, revisions[change.AccountID])
		}
	}
	if err := validateOpenAIOAuthGrantUsers(ctx, tx, changes); err != nil {
		return err
	}

	for _, change := range changes {
		if change.ExpectedRevision == 0 {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO openai_oauth_account_access_policies
					(account_id, mode, default_for_new_users, revision)
				VALUES ($1, $2, $3, 1)
			`, change.AccountID, change.Mode, change.DefaultForNewUsers)
		} else {
			_, err = tx.ExecContext(ctx, `
				UPDATE openai_oauth_account_access_policies
				SET mode = $2,
					default_for_new_users = $3,
					revision = revision + 1,
					updated_at = NOW()
				WHERE account_id = $1
			`, change.AccountID, change.Mode, change.DefaultForNewUsers)
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM openai_oauth_account_user_grants WHERE account_id = $1",
			change.AccountID,
		); err != nil {
			return err
		}
		if len(change.GrantedUserIDs) > 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO openai_oauth_account_user_grants (account_id, user_id)
				SELECT $1, user_id
				FROM unnest($2::bigint[]) AS user_id
			`, change.AccountID, pq.Array(change.GrantedUserIDs)); err != nil {
				return err
			}
		}
	}

	affectedIDs, err := listAffectedOpenAIOAuthAccountIDs(ctx, tx, accountIDs)
	if err != nil {
		return err
	}
	for _, accountID := range affectedIDs {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.refreshSchedulerAccounts(ctx, affectedIDs)
	return nil
}

func validateOpenAIOAuthGrantUsers(ctx context.Context, tx *sql.Tx, changes []service.OpenAIOAuthAccessPolicyChange) error {
	grantSet := make(map[int64]struct{})
	for _, change := range changes {
		for _, userID := range change.GrantedUserIDs {
			grantSet[userID] = struct{}{}
		}
	}
	if len(grantSet) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(grantSet))
	for userID := range grantSet {
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	var validCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users
		WHERE id = ANY($1)
		  AND role IN ('user', 'admin')
		  AND deleted_at IS NULL
	`, pq.Array(userIDs)).Scan(&validCount); err != nil {
		return err
	}
	if validCount != len(userIDs) {
		return infraerrors.BadRequest("OPENAI_OAUTH_ACCESS_INVALID_USERS", "one or more granted users do not exist")
	}
	return nil
}

func listAffectedOpenAIOAuthAccountIDs(ctx context.Context, tx *sql.Tx, rootIDs []int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM accounts
		WHERE deleted_at IS NULL
		  AND (id = ANY($1) OR parent_account_id = ANY($1))
		ORDER BY id
	`, pq.Array(rootIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *openAIOAuthUserAccessRepository) refreshSchedulerAccounts(ctx context.Context, accountIDs []int64) {
	if r.accountRepo == nil || r.schedulerCache == nil {
		return
	}
	for _, accountID := range accountIDs {
		account, err := r.accountRepo.GetByID(ctx, accountID)
		if err == nil && account != nil {
			err = r.schedulerCache.SetAccount(ctx, account)
		}
		if err != nil {
			slog.Warn("failed to refresh scheduler account after OAuth user access update",
				"account_id", accountID,
				"error", fmt.Sprintf("%v", err),
			)
		}
	}
}
