package currency

import "context"

// Registry looks up supported currencies. Get reports false for unknown or inactive codes.
type Registry interface {
	Get(code Code) (Currency, bool)
	Reload(ctx context.Context) error
}
