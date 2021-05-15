package parser

import (
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/planner"
	"github.com/tie/genji-release-test/sql/scanner"
	"github.com/tie/genji-release-test/stream"
)

// parseUpdateStatement parses a update string and returns a Statement AST object.
// This function assumes the UPDATE token has already been consumed.
func (p *Parser) parseUpdateStatement() (*planner.Statement, error) {
	var cfg updateConfig
	var err error

	// Parse table name
	cfg.TableName, err = p.parseIdent()
	if err != nil {
		pErr := err.(*ParseError)
		pErr.Expected = []string{"table_name"}
		return nil, pErr
	}

	// Parse clause: SET or UNSET.
	tok, pos, lit := p.ScanIgnoreWhitespace()
	switch tok {
	case scanner.SET:
		cfg.SetPairs, err = p.parseSetClause()
	case scanner.UNSET:
		cfg.UnsetFields, err = p.parseUnsetClause()
	default:
		err = newParseError(scanner.Tokstr(tok, lit), []string{"SET", "UNSET"}, pos)
	}
	if err != nil {
		return nil, err
	}

	// Parse condition: "WHERE EXPR".
	cfg.WhereExpr, err = p.parseCondition()
	if err != nil {
		return nil, err
	}

	return cfg.ToStream(), nil
}

// parseSetClause parses the "SET" clause of the query.
func (p *Parser) parseSetClause() ([]updateSetPair, error) {
	var pairs []updateSetPair

	firstPair := true
	for {
		if !firstPair {
			// Scan for a comma.
			tok, _, _ := p.ScanIgnoreWhitespace()
			if tok != scanner.COMMA {
				p.Unscan()
				break
			}
		}

		// Scan the identifier for the path name.
		path, err := p.parsePath()
		if err != nil {
			pErr := err.(*ParseError)
			pErr.Expected = []string{"path"}
			return nil, pErr
		}

		// Scan the eq sign
		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.EQ {
			return nil, newParseError(scanner.Tokstr(tok, lit), []string{"="}, pos)
		}

		// Scan the expr for the value.
		expr, _, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, updateSetPair{path, expr})

		firstPair = false
	}

	return pairs, nil
}

func (p *Parser) parseUnsetClause() ([]string, error) {
	var fields []string

	firstField := true
	for {
		if !firstField {
			// Scan for a comma.
			tok, _, _ := p.ScanIgnoreWhitespace()
			if tok != scanner.COMMA {
				p.Unscan()
				break
			}
		}

		// Scan the identifier for the path to unset.
		tok, pos, lit := p.ScanIgnoreWhitespace()
		if tok != scanner.IDENT {
			return nil, newParseError(scanner.Tokstr(tok, lit), []string{"identifier"}, pos)
		}
		fields = append(fields, lit)

		firstField = false
	}
	return fields, nil
}

// UpdateConfig holds UPDATE configuration.
type updateConfig struct {
	TableName string

	// SetPairs is used along with the Set clause. It holds
	// each path with its corresponding value that
	// should be set in the document.
	SetPairs []updateSetPair

	// UnsetFields is used along with the Unset clause. It holds
	// each path that should be unset from the document.
	UnsetFields []string

	WhereExpr expr.Expr
}

type updateSetPair struct {
	path document.Path
	e    expr.Expr
}

// ToTree turns the statement into a stream.
func (cfg updateConfig) ToStream() *planner.Statement {
	s := stream.New(stream.SeqScan(cfg.TableName))

	if cfg.WhereExpr != nil {
		s = s.Pipe(stream.Filter(cfg.WhereExpr))
	}

	if cfg.SetPairs != nil {
		for _, pair := range cfg.SetPairs {
			s = s.Pipe(stream.Set(pair.path, pair.e))
		}
	} else if cfg.UnsetFields != nil {
		for _, name := range cfg.UnsetFields {
			s = s.Pipe(stream.Unset(name))
		}
	}

	s = s.Pipe(stream.TableReplace(cfg.TableName))

	return &planner.Statement{
		Stream:   s,
		ReadOnly: false,
	}
}
