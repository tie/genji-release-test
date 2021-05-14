package planner

import (
	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/query"
	"github.com/tie/genji-release-test/stream"
)

// Statement is a query.Statement using a Stream.
type Statement struct {
	Stream   *stream.Stream
	ReadOnly bool
}

// Run returns a result containing the stream. The stream will be executed by calling the Iterate method of
// the result.
func (s *Statement) Run(tx *database.Transaction, params []expr.Param) (query.Result, error) {
	st, err := Optimize(s.Stream, tx, params)
	if err != nil || st == nil {
		return query.Result{}, err
	}

	return query.Result{
		Iterator: &statementIterator{
			Stream: st,
			Tx:     tx,
			Params: params,
		},
	}, nil
}

// IsReadOnly reports whether the stream will modify the database or only read it.
func (s *Statement) IsReadOnly() bool {
	return s.ReadOnly
}

func (s *Statement) String() string {
	return s.Stream.String()
}

type statementIterator struct {
	Stream *stream.Stream
	Tx     *database.Transaction
	Params []expr.Param
}

func (s *statementIterator) Iterate(fn func(d document.Document) error) error {
	env := expr.Environment{
		Tx:     s.Tx,
		Params: s.Params,
	}

	err := s.Stream.Iterate(&env, func(env *expr.Environment) error {
		// if there is no doc in this specific environment,
		// the last operator is not outputting anything
		// worth returning to the user.
		if env.Doc == nil {
			return nil
		}

		return fn(env.Doc)
	})
	if err == stream.ErrStreamClosed {
		err = nil
	}
	return err
}
