package reconciler

import "context"

// Reconcilable performs one reconciliation pass and returns.
// The caller owns the poll loop.
type Reconcilable interface {
	RunOnce(context.Context) error
}
