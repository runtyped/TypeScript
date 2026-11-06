package loader_test

import (
	"os"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/transformers/runtyped"
)

// The runtyped transformers are off by default in test binaries so that
// upstream tests aren't affected; the loader's tests need them on.
func TestMain(m *testing.M) {
	runtyped.SetEnabled(true)
	os.Exit(m.Run())
}
