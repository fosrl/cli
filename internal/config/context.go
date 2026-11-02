package config

import "context"

type accountStoreCtxKeyType string

const accountStoreCtxKey accountStoreCtxKeyType = "accountStore"

type configCtxKeyType string

const configCtxKey configCtxKeyType = "configuration"

func WithAccountStore(ctx context.Context, store *AccountStore) context.Context {
	return context.WithValue(ctx, accountStoreCtxKey, store)
}

// LookupAccountStore returns the account store if one was set up, for
// commands that work without an account.
func LookupAccountStore(ctx context.Context) (*AccountStore, bool) {
	store, ok := ctx.Value(accountStoreCtxKey).(*AccountStore)
	return store, ok
}

func AccountStoreFromContext(ctx context.Context) *AccountStore {
	logger, ok := ctx.Value(accountStoreCtxKey).(*AccountStore)
	if !ok {
		panic("accountStore not present in context")
	}
	return logger
}

func WithConfig(ctx context.Context, store *Config) context.Context {
	return context.WithValue(ctx, configCtxKey, store)
}

func ConfigFromContext(ctx context.Context) *Config {
	logger, ok := ctx.Value(configCtxKey).(*Config)
	if !ok {
		panic("configuration not present in context")
	}
	return logger
}
