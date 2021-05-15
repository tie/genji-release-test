package planner

import (
	"errors"

	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/query"
	"github.com/tie/genji-release-test/stream"
)

// ExplainStmt is a query.Statement that
// displays information about how a statement
// is going to be executed, without executing it.
type ExplainStmt struct {
	Statement query.Statement
}

// Run analyses the inner statement and displays its execution plan.
// If the statement is a stream, Optimize will be called prior to
// displaying all the operations.
// Explain currently only works on SELECT, UPDATE, INSERT and DELETE statements.
func (s *ExplainStmt) Run(tx *database.Transaction, params []expr.Param) (query.Result, error) {
	switch t := s.Statement.(type) {
	case *Statement:
		s, err := Optimize(t.Stream, tx, params)
		if err != nil {
			return query.Result{}, err
		}

		var plan string
		if s != nil {
			plan = s.String()
		} else {
			plan = "<no exec>"
		}

		newStatement := Statement{
			Stream: &stream.Stream{
				Op: stream.Project(
					&expr.NamedExpr{
						ExprName: "plan",
						Expr:     expr.LiteralValue(document.NewTextValue(plan)),
					}),
			},
			ReadOnly: true,
		}
		return newStatement.Run(tx, params)
	}

	return query.Result{}, errors.New("EXPLAIN only works on INSERT, SELECT, UPDATE AND DELETE statements")
}

// IsReadOnly indicates that this statement doesn't write anything into
// the database.
func (s *ExplainStmt) IsReadOnly() bool {
	return true
}
