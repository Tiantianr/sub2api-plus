package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthUserAccessRepositoryListUsersOrdersByIDDescending(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`(?s)FROM users AS u.*ORDER BY u\.id DESC`).
		WithArgs("", "").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "status", "api_key_group_ids", "subscription_group_ids", "account_ids",
		}).
			AddRow(int64(20), "newer@example.com", "active", "{}", "{}", "{}").
			AddRow(int64(10), "older@example.com", "active", "{}", "{}", "{}"))

	repo := &openAIOAuthUserAccessRepository{db: db}
	users, err := repo.ListUsers(context.Background(), "", "")
	require.NoError(t, err)
	require.Equal(t, []int64{20, 10}, []int64{users[0].ID, users[1].ID})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpenAIOAuthUserAccessRepositoryListAccountsIncludesAPIKeys(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`(?s)WHERE a\.platform = 'openai'\s+AND a\.type IN \('oauth', 'apikey'\)`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "type", "status", "extra", "group_ids", "mode", "default_for_new_users", "revision", "user_ids",
		}).
			AddRow(int64(1), "OpenAI OAuth", "oauth", "active", []byte("{}"), "{1,2}", "public", false, int64(0), "{}").
			AddRow(int64(2), "OpenAI APIKey", "apikey", "active", []byte("{}"), "{2,3}", "restricted", true, int64(3), "{101,102}"))

	repo := &openAIOAuthUserAccessRepository{db: db}
	accounts, err := repo.ListAccounts(context.Background())
	require.NoError(t, err)
	require.Len(t, accounts, 2)

	require.Equal(t, int64(1), accounts[0].ID)
	require.Equal(t, "oauth", accounts[0].Type)
	require.Equal(t, "public", accounts[0].Mode)

	require.Equal(t, int64(2), accounts[1].ID)
	require.Equal(t, "apikey", accounts[1].Type)
	require.Equal(t, "restricted", accounts[1].Mode)
	require.True(t, accounts[1].DefaultForNewUsers)
	require.Equal(t, int64(3), accounts[1].Revision)
	require.Equal(t, []int64{101, 102}, accounts[1].GrantedUserIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}
