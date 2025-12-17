package accounts

import "context"

type accountStoreCtxKeyType string

const accountStoreCtxKey accountStoreCtxKeyType = "accountStore"

func WithAccountStore(ctx context.Context, store *AccountStore) context.Context {
	return context.WithValue(ctx, accountStoreCtxKey, store)
}

func FromContext(ctx context.Context) *AccountStore {
	logger, ok := ctx.Value(accountStoreCtxKey).(*AccountStore)
	if !ok {
		panic("accountStore not present in context")
	}
	return logger
}
