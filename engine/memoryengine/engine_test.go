package memoryengine_test

import (
	"testing"

	"github.com/tie/genji-release-test/engine"
	"github.com/tie/genji-release-test/engine/enginetest"
	"github.com/tie/genji-release-test/engine/memoryengine"
)

func builder() (engine.Engine, func()) {
	ng := memoryengine.NewEngine()
	return ng, func() { ng.Close() }
}

func TestMemoryEngine(t *testing.T) {
	enginetest.TestSuite(t, builder)
}

func BenchmarkMemoryEngineStorePut(b *testing.B) {
	enginetest.BenchmarkStorePut(b, builder)
}

func BenchmarkMemoryEngineStoreScan(b *testing.B) {
	enginetest.BenchmarkStoreScan(b, builder)
}
