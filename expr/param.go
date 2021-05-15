package expr

import (
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/stringutil"
)

// A Param represents a parameter passed by the user to the statement.
type Param struct {
	// Name of the param
	Name string

	// Value is the parameter value.
	Value interface{}
}

// NamedParam is an expression which represents the name of a parameter.
type NamedParam string

// Eval looks up for the parameters in the env for the one that has the same name as p
// and returns the value.
func (p NamedParam) Eval(env *Environment) (document.Value, error) {
	return env.GetParamByName(string(p))
}

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (p NamedParam) IsEqual(other Expr) bool {
	o, ok := other.(NamedParam)
	return ok && p == o
}

// String implements the stringutil.Stringer interface.
func (p NamedParam) String() string {
	return stringutil.Sprintf("$%s", string(p))
}

// PositionalParam is an expression which represents the position of a parameter.
type PositionalParam int

// Eval looks up for the parameters in the env for the one that is has the same position as p
// and returns the value.
func (p PositionalParam) Eval(env *Environment) (document.Value, error) {
	return env.GetParamByIndex(int(p))
}

// IsEqual compares this expression with the other expression and returns
// true if they are equal.
func (p PositionalParam) IsEqual(other Expr) bool {
	o, ok := other.(PositionalParam)
	return ok && p == o
}

// String implements the stringutil.Stringer interface.
func (p PositionalParam) String() string {
	return "?"
}
