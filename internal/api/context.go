package api

import "context"

type apiClientCtxKeyType string

const apiClientCtxKey apiClientCtxKeyType = "apiClient"

func WithAPIClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, apiClientCtxKey, client)
}

func FromContext(ctx context.Context) *Client {
	logger, ok := ctx.Value(apiClientCtxKey).(*Client)
	if !ok {
		panic("apiClient not present in context")
	}
	return logger
}
