package parser_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/planner"
	"github.com/tie/genji-release-test/query"
	"github.com/tie/genji-release-test/sql/parser"
	"github.com/tie/genji-release-test/stream"
)

func TestParserMultiStatement(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected []query.Statement
	}{
		{"OnlyCommas", ";;;", nil},
		{"TrailingComma", "SELECT * FROM foo;;;DELETE FROM foo;", []query.Statement{
			&planner.Statement{
				Stream:   stream.New(stream.SeqScan("foo")).Pipe(stream.Project(expr.Wildcard{})),
				ReadOnly: true,
			},
			&planner.Statement{
				Stream: stream.New(stream.SeqScan("foo")).Pipe(stream.TableDelete("foo")),
			},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := parser.ParseQuery(test.s)
			require.NoError(t, err)
			require.EqualValues(t, test.expected, q.Statements)
		})
	}
}

func TestParserDivideByZero(t *testing.T) {
	// See https://github.com/tie/genji-release-test/issues/268
	require.NotPanics(t, func() {
		_, _ = parser.ParseQuery("SELECT * FROM t LIMIT 0 % .5")
	})
}
