package api

import "context"

type apiClientCtxKeyType string

const apiClientCtxKey apiClientCtxKeyType = "apiClient"

func WithAPIClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, apiClientCtxKey, client)
}

func ClientFromContext(ctx context.Context) (*Client, bool) {
	client, ok := ctx.Value(apiClientCtxKey).(*Client)
	return client, ok
}

func FromContext(ctx context.Context) *Client {
	client, ok := ClientFromContext(ctx)
	if !ok {
		panic("apiClient not present in context")
	}
	return client
}
