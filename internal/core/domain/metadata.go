package domain

import "context"

type metadataContextKey struct{}

// ClientMetadata stores client forensic information captured from HTTP request headers and connections.
type ClientMetadata struct {
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	RequestID string `json:"request_id"`
}

// WithClientMetadata injects ClientMetadata into the provided context.
func WithClientMetadata(ctx context.Context, meta ClientMetadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, metadataContextKey{}, meta)
}

// GetClientMetadata retrieves ClientMetadata from the provided context if present.
func GetClientMetadata(ctx context.Context) (ClientMetadata, bool) {
	if ctx == nil {
		return ClientMetadata{}, false
	}
	meta, ok := ctx.Value(metadataContextKey{}).(ClientMetadata)
	return meta, ok
}
