package db

import (
	"context"
	"fmt"
	"kabsa/ent"
)

type TxFunc func(ctx context.Context, tx *ent.Tx) error

// WithTx runs the given function in a transaction.
func (c *Client) WithTx(ctx context.Context, fn TxFunc) error {
	tx, err := c.ent.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start tx: %w", err)
	}

	// If fn returns error, rollback.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}
