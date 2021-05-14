package expr_test

import (
	"testing"

	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
)

func TestPkExpr(t *testing.T) {
	tests := []struct {
		name string
		env  *expr.Environment
		res  document.Value
	}{
		{"empty env", &expr.Environment{}, nullLitteral},
		{"env with doc", envWithDoc, nullLitteral},
		{"env with doc and key", envWithDocAndKey, document.NewIntegerValue(1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testExpr(t, "pk()", test.env, test.res, false)
		})
	}
}
