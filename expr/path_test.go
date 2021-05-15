package expr_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/sql/parser"
	"github.com/stretchr/testify/require"
)

func TestPathExpr(t *testing.T) {
	tests := []struct {
		expr  string
		res   document.Value
		fails bool
	}{
		{"a", document.NewIntegerValue(1), false},
		{"b", func() document.Value {
			fb := document.NewFieldBuffer()
			err := json.Unmarshal([]byte(`{"foo bar": [1, 2]}`), fb)
			require.NoError(t, err)
			return document.NewDocumentValue(fb)
		}(),
			false},
		{"b.`foo bar`[0]", document.NewIntegerValue(1), false},
		{"_v.b.`foo bar`[0]", document.NewNullValue(), false},
		{"b.`foo bar`[1]", document.NewIntegerValue(2), false},
		{"b.`foo bar`[2]", nullLitteral, false},
		{"b[0]", nullLitteral, false},
		{"c[0]", document.NewIntegerValue(1), false},
		{"c[1].foo", document.NewTextValue("bar"), false},
		{"c.foo", nullLitteral, false},
		{"d", nullLitteral, false},
	}

	d := document.NewFromJSON([]byte(`{
		"a": 1,
		"b": {"foo bar": [1, 2]},
		"c": [1, {"foo": "bar"}, [1, 2]]
	}`))

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			testExpr(t, test.expr, expr.NewEnvironment(d), test.res, test.fails)
		})
	}

	t.Run("empty env", func(t *testing.T) {
		testExpr(t, "a", &expr.Environment{}, nullLitteral, true)
	})
}

func TestPathIsEqual(t *testing.T) {
	tests := []struct {
		a, b    string
		isEqual bool
	}{
		{`a`, `a`, true},
		{`a[0].b`, `a[0].b`, true},
		{`a[0].b`, `a[1].b`, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s = %s", test.a, test.b), func(t *testing.T) {
			pa, err := parser.ParsePath(test.a)
			require.NoError(t, err)
			ea := expr.Path(pa)

			pb, err := parser.ParsePath(test.b)
			require.NoError(t, err)
			eb := expr.Path(pb)

			require.Equal(t, test.isEqual, ea.IsEqual(eb))
		})
	}
}

func TestEnvPathExpr(t *testing.T) {
	tests := []struct {
		expr  string
		res   document.Value
		fails bool
	}{
		{"a", document.NewIntegerValue(1), false},
		{"b", func() document.Value {
			fb := document.NewFieldBuffer()
			err := json.Unmarshal([]byte(`{"foo bar": [1, 2]}`), fb)
			require.NoError(t, err)
			return document.NewDocumentValue(fb)
		}(),
			false},
		{"b.`foo bar`[0]", document.NewIntegerValue(1), false},
		{"_v.b.`foo bar`[0]", document.NewNullValue(), false},
		{"b.`foo bar`[1]", document.NewIntegerValue(2), false},
		{"b.`foo bar`[2]", nullLitteral, false},
		{"b[0]", nullLitteral, false},
		{"c[0]", document.NewIntegerValue(1), false},
		{"c[1].foo", document.NewTextValue("bar"), false},
		{"c.foo", nullLitteral, false},
		{"d", nullLitteral, false},
	}

	d := document.NewFromJSON([]byte(`{
		"a": 1,
		"b": {"foo bar": [1, 2]},
		"c": [1, {"foo": "bar"}, [1, 2]]
	}`))

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			testExpr(t, test.expr, expr.NewEnvironment(d), test.res, test.fails)
		})
	}

	t.Run("empty env", func(t *testing.T) {
		testExpr(t, "a", &expr.Environment{}, nullLitteral, true)
	})
}
