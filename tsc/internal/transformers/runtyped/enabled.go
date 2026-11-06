package runtyped

import (
	"sync/atomic"
	"testing"
)

// enabled controls whether the emitter runs the runtyped transformers.
//
// The shipped compiler always reflects. In test binaries the transformers are
// off by default so that upstream TypeScript tests run against the stock emit
// pipeline and their reference baselines stay untouched; runtyped's own test
// packages opt back in with SetEnabled(true) from their TestMain.
var enabled atomic.Bool

func init() {
	enabled.Store(!testing.Testing())
}

// Enabled reports whether the runtyped transformers should be added to emit.
func Enabled() bool {
	return enabled.Load()
}

// SetEnabled turns the runtyped transformers on or off for subsequent emits.
// It is intended for tests.
func SetEnabled(value bool) {
	enabled.Store(value)
}
