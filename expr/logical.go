package expr

import (
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/sql/scanner"
	"github.com/tie/genji-release-test/stringutil"
)

// AndOp is the And operator.
type AndOp struct {
	*simpleOperator
}

// And creates an expression that evaluates a And b And returns true if both are truthy.
func And(a, b Expr) Expr {
	return &AndOp{&simpleOperator{a, b, scanner.AND}}
}

// Eval implements the Expr interface. It evaluates a and b and returns true if both evaluate
// to true.
func (op *AndOp) Eval(env *Environment) (document.Value, error) {
	s, err := op.a.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err := s.IsTruthy()
	if !isTruthy || err != nil {
		return falseLitteral, err
	}

	s, err = op.b.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err = s.IsTruthy()
	if !isTruthy || err != nil {
		return falseLitteral, err
	}

	return trueLitteral, nil
}

// OrOp is the Or operator.
type OrOp struct {
	*simpleOperator
}

// Or creates an expression that first evaluates a, returns true if truthy, then evaluates b, returns true if truthy Or false if falsy.
func Or(a, b Expr) Expr {
	return &OrOp{&simpleOperator{a, b, scanner.OR}}
}

// Eval implements the Expr interface. It evaluates a and b and returns true if a or b evalutate
// to true.
func (op *OrOp) Eval(env *Environment) (document.Value, error) {
	s, err := op.a.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err := s.IsTruthy()
	if err != nil {
		return falseLitteral, err
	}
	if isTruthy {
		return trueLitteral, nil
	}

	s, err = op.b.Eval(env)
	if err != nil {
		return falseLitteral, err
	}
	isTruthy, err = s.IsTruthy()
	if err != nil {
		return falseLitteral, err
	}
	if isTruthy {
		return trueLitteral, nil
	}

	return falseLitteral, nil
}

// NotOp is the NOT unary operator.
type NotOp struct {
	*simpleOperator
}

// Not creates an expression that returns true if e is falsy.
func Not(e Expr) Expr {
	return &NotOp{&simpleOperator{a: e}}
}

// Eval implements the Expr interface. It evaluates e and returns true if b is falsy
func (op *NotOp) Eval(env *Environment) (document.Value, error) {
	s, err := op.a.Eval(env)
	if err != nil {
		return falseLitteral, err
	}

	isTruthy, err := s.IsTruthy()
	if err != nil {
		return falseLitteral, err
	}
	if isTruthy {
		return falseLitteral, nil
	}

	return trueLitteral, nil
}

// String implements the stringutil.Stringer interface.
func (op *NotOp) String() string {
	return stringutil.Sprintf("NOT %v", op.a)
}
