package parser

import (
	"github.com/tie/genji-release-test/query"
	"github.com/tie/genji-release-test/sql/scanner"
)

// parseBeginStatement parses a BEGIN statement.
// This function assumes the BEGIN token has already been consumed.
func (p *Parser) parseBeginStatement() (query.Statement, error) {
	// parse optional TRANSACTION token
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.TRANSACTION {
		p.Unscan()
	}

	// parse optional READ token
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.READ {
		p.Unscan()
		return query.BeginStmt{Writable: true}, nil
	}

	// parse ONLY token
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == scanner.ONLY {
		return query.BeginStmt{Writable: false}, nil
	}

	p.Unscan()

	// parse WRITE token
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.WRITE {
		return query.BeginStmt{}, newParseError(scanner.Tokstr(tok, lit), []string{"ONLY", "WRITE"}, pos)
	}

	return query.BeginStmt{Writable: true}, nil
}

// parseRollbackStatement parses a ROLLBACK statement.
// This function assumes the ROLLBACK token has already been consumed.
func (p *Parser) parseRollbackStatement() (query.Statement, error) {
	// parse optional TRANSACTION token
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.TRANSACTION {
		p.Unscan()
	}

	return query.RollbackStmt{}, nil
}

// parseCommitStatement parses a COMMIT statement.
// This function assumes the COMMIT token has already been consumed.
func (p *Parser) parseCommitStatement() (query.Statement, error) {
	// parse optional TRANSACTION token
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.TRANSACTION {
		p.Unscan()
	}

	return query.CommitStmt{}, nil
}
