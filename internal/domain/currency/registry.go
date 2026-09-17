package currency

import "context"

type Registry interface {
	Get(code Code) (Currency, bool)
	Reload(ctx context.Context) error
}
