package stream_test

import (
	"fmt"
	"testing"

	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/sql/parser"
	"github.com/tie/genji-release-test/stream"
	"github.com/tie/genji-release-test/testutil"
	"github.com/stretchr/testify/require"
)

func TestStream(t *testing.T) {
	s := stream.New(stream.Documents(
		testutil.MakeDocument(t, `{"a": 1}`),
		testutil.MakeDocument(t, `{"a": 2}`),
	))

	s = s.Pipe(stream.Map(parser.MustParseExpr("{a: a + 1}")))
	s = s.Pipe(stream.Filter(parser.MustParseExpr("a > 2")))

	var count int64
	err := s.Iterate(new(expr.Environment), func(env *expr.Environment) error {
		d, ok := env.GetDocument()
		require.True(t, ok)
		require.JSONEq(t, fmt.Sprintf(`{"a": %d}`, count+3), document.NewDocumentValue(d).String())
		count++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}
