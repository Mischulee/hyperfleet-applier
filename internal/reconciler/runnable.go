// Package reconciler defines the two lifecycles shared by the applier.
package reconciler

import "context"

// Runnable owns a loop that runs until ctx is canceled.
// A caller-driven stop is successful and must return nil.
type Runnable interface {
	Start(context.Context) error
}
