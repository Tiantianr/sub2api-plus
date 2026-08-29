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
