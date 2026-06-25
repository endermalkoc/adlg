package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// WithRetryTx runs fn inside a transaction on the pinned conn, retrying on
// serialization failures (deadlock 1213 / lock-wait timeout 1205) and pre-commit
// connection errors. It does NOT retry after tx.Commit() — a commit-phase failure
// is ambiguous (the write may have landed), and replaying could double-apply.
//
// conn must be a single dedicated connection (db.Conn): branch state is
// connection-scoped, so the caller checks out the target branch on this same conn
// before calling, and runs the Dolt commit on it afterward.
func WithRetryTx(ctx context.Context, conn *sql.Conn, fn func(tx *sql.Tx) error) error {
	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff(ctx, attempt)
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return err
		}
		if err := fn(tx); err != nil {
			_ = tx.Rollback()
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return err
		}
		if err := tx.Commit(); err != nil {
			// Commit phase: do not retry (ambiguous double-apply).
			return fmt.Errorf("commit: %w", err)
		}
		return nil
	}
	return fmt.Errorf("gave up after %d attempts: %w", maxAttempts, lastErr)
}

// isRetryable reports whether err is a Dolt/MySQL serialization failure safe to
// replay (the server guarantees rollback on these).
func isRetryable(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1213 || me.Number == 1205
	}
	return false
}

func backoff(ctx context.Context, attempt int) {
	d := time.Duration(25*(1<<attempt)) * time.Millisecond
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
