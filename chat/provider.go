package chat

import "context"

// ChatProvider defines the interface for LLM provider implementations.
type ChatProvider interface {
	Invoke(ctx context.Context, fullInput string) (string, float64, error)
	Name() string
}
