package custom

import (
	"testing"

	"github.com/tie/genji-release-test/document/encoding"
	"github.com/tie/genji-release-test/document/encoding/encodingtest"
)

func TestCodec(t *testing.T) {
	encodingtest.TestCodec(t, func() encoding.Codec {
		return NewCodec()
	})
}

func BenchmarkCodec(b *testing.B) {
	encodingtest.BenchmarkCodec(b, func() encoding.Codec {
		return NewCodec()
	})
}
