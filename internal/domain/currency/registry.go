package currency

import "context"

// Registry looks up supported currencies. Get reports false for unknown or
// inactive codes; List returns the active currencies sorted by code.
type Registry interface {
	Get(code Code) (Currency, bool)
	List() []Currency
	Reload(ctx context.Context) error
}
