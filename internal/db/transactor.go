package db

import "context"

type Transactor interface {
	WithTx(ctx context.Context, fn TxFunc) error
}
