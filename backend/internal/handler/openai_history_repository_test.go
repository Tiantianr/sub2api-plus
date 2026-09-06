package handler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type openAIHistoryHandlerRepository struct {
	service.AccountRepository
	mu         sync.Mutex
	bindings   map[string]service.OpenAIConversationBinding
	lookups    int
	writes     int
	failWrites string
}

func withOpenAIHistoryTestRepository(repo service.AccountRepository) service.AccountRepository {
	if repo == nil {
		return nil
	}
	if _, ok := repo.(service.OpenAIConversationBindingRepository); ok {
		return repo
	}
	return &openAIHistoryHandlerRepository{AccountRepository: repo, bindings: make(map[string]service.OpenAIConversationBinding)}
}

func handlerHistoryKey(user, group int64, kind, key string) string {
	return fmt.Sprintf("%d:%d:%s:%s", user, group, kind, key)
}

func (r *openAIHistoryHandlerRepository) GetOpenAIConversationBinding(_ context.Context, user, group int64, kind, key string) (*service.OpenAIConversationBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lookups++
	binding, found := r.bindings[handlerHistoryKey(user, group, kind, key)]
	if !found {
		return nil, nil
	}
	return &binding, nil
}

func (r *openAIHistoryHandlerRepository) SaveOpenAIConversationBinding(_ context.Context, binding *service.OpenAIConversationBinding, expectedRevision int64) (*service.OpenAIConversationBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes++
	if binding.Type == r.failWrites {
		return nil, errors.New("test ownership storage unavailable")
	}
	key := handlerHistoryKey(binding.UserID, binding.ScopeGroupID, binding.Type, binding.Key)
	old, exists := r.bindings[key]
	if exists && !(binding.Type == "session" && old.Revision == expectedRevision) &&
		!(old.AccountID == binding.AccountID && old.CredentialOwnerAccountID == binding.CredentialOwnerAccountID &&
			old.OAuthAccountID == binding.OAuthAccountID && old.OAuthUserID == binding.OAuthUserID && old.PolicyScopeVersion == binding.PolicyScopeVersion) {
		return nil, service.ErrOpenAIHistoryConflict
	}
	saved := *binding
	saved.Revision, saved.Valid = old.Revision+1, true
	r.bindings[key] = saved
	return &saved, nil
}
