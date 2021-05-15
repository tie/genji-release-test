package parser

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/expr"
	"github.com/tie/genji-release-test/sql/scanner"
	"github.com/tie/genji-release-test/stringutil"
)

type dummyOperator struct {
	rightHand expr.Expr
}

func (d *dummyOperator) Token() scanner.Token                           { panic("not implemented") }
func (d *dummyOperator) Equal(expr.Expr) bool                           { panic("not implemented") }
func (d *dummyOperator) Eval(*expr.Environment) (document.Value, error) { panic("not implemented") }
func (d *dummyOperator) String() string                                 { panic("not implemented") }
func (d *dummyOperator) Precedence() int                                { panic("not implemented") }
func (d *dummyOperator) LeftHand() expr.Expr                            { panic("not implemented") }
func (d *dummyOperator) RightHand() expr.Expr                           { return d.rightHand }
func (d *dummyOperator) SetLeftHandExpr(e expr.Expr)                    { panic("not implemented") }
func (d *dummyOperator) SetRightHandExpr(e expr.Expr)                   { d.rightHand = e }

// ParseExpr parses an expression.
func (p *Parser) ParseExpr() (e expr.Expr, lit string, err error) {
	return p.parseExprWithMinPrecedence(0)
}

func (p *Parser) parseExprWithMinPrecedence(precedence int) (e expr.Expr, lit string, err error) {
	// enable the expression buffer to store the literal representation
	// of the parsed expression
	if p.buf == nil {
		p.buf = new(bytes.Buffer)
		defer func() { p.buf = nil }()
	}

	// Dummy root node.
	var root expr.Operator = new(dummyOperator)

	// Parse a non-binary expression type to start.
	// This variable will always be the root of the expression tree.
	e, err = p.parseUnaryExpr()
	if err != nil {
		return nil, "", err
	}
	root.SetRightHandExpr(e)

	// Loop over operations and unary exprs and build a tree based on precedence.
	for {
		// If the next token is NOT an operator then return the expression.
		op, tok, err := p.parseOperator(precedence)
		if err != nil {
			return nil, "", err
		}
		if tok == 0 {
			return root.RightHand(), strings.TrimSpace(p.buf.String()), nil
		}

		var rhs expr.Expr

		if rhs, err = p.parseUnaryExpr(); err != nil {
			return nil, "", err
		}

		// Find the right spot in the tree to add the new expression by
		// descending the RHS of the expression tree until we reach the last
		// BinaryExpr or a BinaryExpr whose RHS has an operator with
		// precedence >= the operator being added.
		for node := root.(expr.Operator); ; {
			p, ok := node.RightHand().(expr.Operator)
			if !ok || p.Precedence() >= tok.Precedence() {
				// Add the new expression here and break.
				node.SetRightHandExpr(op(node.RightHand(), rhs))
				break
			}
			node = p
		}
	}
}

func (p *Parser) parseOperator(minPrecedence int) (func(lhs, rhs expr.Expr) expr.Expr, scanner.Token, error) {
	op, _, _ := p.ScanIgnoreWhitespace()
	if !op.IsOperator() && op != scanner.NOT {
		p.Unscan()
		return nil, 0, nil
	}

	// Ignore currently unused operators.
	if op == scanner.EQREGEX || op == scanner.NEQREGEX {
		p.Unscan()
		return nil, 0, nil
	}

	switch {
	case op == scanner.EQ && op.Precedence() >= minPrecedence:
		return expr.Eq, op, nil
	case op == scanner.NEQ && op.Precedence() >= minPrecedence:
		return expr.Neq, op, nil
	case op == scanner.GT && op.Precedence() >= minPrecedence:
		return expr.Gt, op, nil
	case op == scanner.GTE && op.Precedence() >= minPrecedence:
		return expr.Gte, op, nil
	case op == scanner.LT && op.Precedence() >= minPrecedence:
		return expr.Lt, op, nil
	case op == scanner.LTE && op.Precedence() >= minPrecedence:
		return expr.Lte, op, nil
	case op == scanner.AND && op.Precedence() >= minPrecedence:
		return expr.And, op, nil
	case op == scanner.OR && op.Precedence() >= minPrecedence:
		return expr.Or, op, nil
	case op == scanner.ADD && op.Precedence() >= minPrecedence:
		return expr.Add, op, nil
	case op == scanner.SUB && op.Precedence() >= minPrecedence:
		return expr.Sub, op, nil
	case op == scanner.MUL && op.Precedence() >= minPrecedence:
		return expr.Mul, op, nil
	case op == scanner.DIV && op.Precedence() >= minPrecedence:
		return expr.Div, op, nil
	case op == scanner.MOD && op.Precedence() >= minPrecedence:
		return expr.Mod, op, nil
	case op == scanner.BITWISEAND && op.Precedence() >= minPrecedence:
		return expr.BitwiseAnd, op, nil
	case op == scanner.BITWISEOR && op.Precedence() >= minPrecedence:
		return expr.BitwiseOr, op, nil
	case op == scanner.BITWISEXOR && op.Precedence() >= minPrecedence:
		return expr.BitwiseXor, op, nil
	case op == scanner.IN && op.Precedence() >= minPrecedence:
		return expr.In, op, nil
	case op == scanner.IS && op.Precedence() >= minPrecedence:
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok == scanner.NOT {
			return expr.IsNot, op, nil
		}
		p.Unscan()
		return expr.Is, op, nil
	case op == scanner.NOT:
		tok, pos, lit := p.ScanIgnoreWhitespace()
		switch {
		case tok == scanner.IN && tok.Precedence() >= minPrecedence:
			return expr.NotIn, op, nil
		case tok == scanner.LIKE && tok.Precedence() >= minPrecedence:
			return expr.NotLike, op, nil
		}

		return nil, 0, newParseError(scanner.Tokstr(tok, lit), []string{"IN, LIKE"}, pos)
	case op == scanner.LIKE && op.Precedence() >= minPrecedence:
		return expr.Like, op, nil
	case op == scanner.CONCAT && op.Precedence() >= minPrecedence:
		return expr.Concat, op, nil
	case op == scanner.BETWEEN && op.Precedence() >= minPrecedence:
		a, _, err := p.parseExprWithMinPrecedence(op.Precedence())
		if err != nil {
			return nil, op, err
		}
		err = p.parseTokens(scanner.AND)
		if err != nil {
			return nil, op, err
		}

		return expr.Between(a), op, nil
	}

	p.Unscan()

	return nil, 0, nil
}

// parseUnaryExpr parses an non-binary expression.
func (p *Parser) parseUnaryExpr() (expr.Expr, error) {
	tok, pos, lit := p.ScanIgnoreWhitespace()
	switch tok {
	case scanner.CAST:
		p.Unscan()
		return p.parseCastExpression()
	case scanner.IDENT:
		// if the next token is a left parenthesis, this is a function
		if tok1, _, _ := p.Scan(); tok1 == scanner.LPAREN {
			p.Unscan()
			p.Unscan()
			return p.parseFunction()
		}
		p.Unscan()
		p.Unscan()
		field, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		fs := expr.Path(field)
		return fs, nil
	case scanner.NAMEDPARAM:
		if len(lit) == 1 {
			return nil, &ParseError{Message: "missing param name"}
		}
		if p.orderedParams > 0 {
			return nil, &ParseError{Message: "cannot mix positional arguments with named arguments"}
		}
		p.namedParams++
		return expr.NamedParam(lit[1:]), nil
	case scanner.POSITIONALPARAM:
		if p.namedParams > 0 {
			return nil, &ParseError{Message: "cannot mix positional arguments with named arguments"}
		}
		p.orderedParams++
		return expr.PositionalParam(p.orderedParams), nil
	case scanner.STRING:
		return expr.LiteralValue(document.NewTextValue(lit)), nil
	case scanner.NUMBER:
		v, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			return nil, &ParseError{Message: "unable to parse number", Pos: pos}
		}
		return expr.LiteralValue(document.NewDoubleValue(v)), nil
	case scanner.INTEGER:
		v, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			// The literal may be too large to fit into an int64, parse as Float64
			if v, err := strconv.ParseFloat(lit, 64); err == nil {
				return expr.LiteralValue(document.NewDoubleValue(v)), nil
			}
			return nil, &ParseError{Message: "unable to parse integer", Pos: pos}
		}
		return expr.LiteralValue(document.NewIntegerValue(v)), nil
	case scanner.TRUE, scanner.FALSE:
		return expr.LiteralValue(document.NewBoolValue(tok == scanner.TRUE)), nil
	case scanner.NULL:
		return expr.LiteralValue(document.NewNullValue()), nil
	case scanner.LBRACKET:
		p.Unscan()
		e, err := p.ParseDocument()
		return e, err
	case scanner.LSBRACKET:
		p.Unscan()
		return p.parseExprList(scanner.LSBRACKET, scanner.RSBRACKET)
	case scanner.LPAREN:
		e, _, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}

		tok, pos, lit := p.ScanIgnoreWhitespace()
		switch tok {
		case scanner.RPAREN:
			return expr.Parentheses{E: e}, nil
		case scanner.COMMA:
			exprList, err := p.parseExprListUntil(scanner.RPAREN)
			if err != nil {
				return nil, err
			}

			// prepend first parsed expression
			exprList = append([]expr.Expr{e}, exprList...)
			return exprList, nil
		}

		return nil, newParseError(scanner.Tokstr(tok, lit), []string{")", ","}, pos)
	case scanner.NOT:
		e, _, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		return expr.Not(e), nil
	default:
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"identifier", "string", "number", "bool"}, pos)
	}
}

// parseIdent parses an identifier.
func (p *Parser) parseIdent() (string, error) {
	tok, pos, lit := p.ScanIgnoreWhitespace()
	if tok != scanner.IDENT {
		return "", newParseError(scanner.Tokstr(tok, lit), []string{"identifier"}, pos)
	}

	return lit, nil
}

// parseIdentList parses a comma delimited list of identifiers.
func (p *Parser) parseIdentList() ([]string, error) {
	// Parse first (required) identifier.
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}
	idents := []string{ident}

	// Parse remaining (optional) identifiers.
	for {
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.COMMA {
			p.Unscan()
			return idents, nil
		}

		if ident, err = p.parseIdent(); err != nil {
			return nil, err
		}

		idents = append(idents, ident)
	}
}

// parseParam parses a positional or named param.
func (p *Parser) parseParam() (expr.Expr, error) {
	tok, _, lit := p.ScanIgnoreWhitespace()
	switch tok {
	case scanner.NAMEDPARAM:
		if len(lit) == 1 {
			return nil, &ParseError{Message: "missing param name"}
		}
		if p.orderedParams > 0 {
			return nil, &ParseError{Message: "cannot mix positional arguments with named arguments"}
		}
		p.namedParams++
		return expr.NamedParam(lit[1:]), nil
	case scanner.POSITIONALPARAM:
		if p.namedParams > 0 {
			return nil, &ParseError{Message: "cannot mix positional arguments with named arguments"}
		}
		p.orderedParams++
		return expr.PositionalParam(p.orderedParams), nil
	default:
		return nil, nil
	}
}

func (p *Parser) parseType() (document.ValueType, error) {
	tok, pos, lit := p.ScanIgnoreWhitespace()
	switch tok {
	case scanner.TYPEARRAY:
		return document.ArrayValue, nil
	case scanner.TYPEBLOB:
		return document.BlobValue, nil
	case scanner.TYPEBOOL:
		return document.BoolValue, nil
	case scanner.TYPEBYTES:
		return document.BlobValue, nil
	case scanner.TYPEDOCUMENT:
		return document.DocumentValue, nil
	case scanner.TYPEREAL:
		return document.DoubleValue, nil
	case scanner.TYPEDOUBLE:
		tok, _, _ := p.ScanIgnoreWhitespace()
		if tok == scanner.PRECISION {
			return document.DoubleValue, nil
		}
		p.Unscan()
		return document.DoubleValue, nil
	case scanner.TYPEINTEGER, scanner.TYPEINT, scanner.TYPEINT2, scanner.TYPEINT8, scanner.TYPETINYINT,
		scanner.TYPEBIGINT, scanner.TYPEMEDIUMINT, scanner.TYPESMALLINT:
		return document.IntegerValue, nil
	case scanner.TYPETEXT:
		return document.TextValue, nil
	case scanner.TYPEVARCHAR, scanner.TYPECHARACTER:
		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.LPAREN {
			return 0, newParseError(scanner.Tokstr(tok, lit), []string{"("}, pos)
		}

		// The value between parentheses is not used.
		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.INTEGER {
			return 0, newParseError(scanner.Tokstr(tok, lit), []string{"integer"}, pos)
		}

		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.RPAREN {
			return 0, newParseError(scanner.Tokstr(tok, lit), []string{")"}, pos)
		}

		return document.TextValue, nil
	}

	return 0, newParseError(scanner.Tokstr(tok, lit), []string{"type"}, pos)
}

// ParseDocument parses a document
func (p *Parser) ParseDocument() (*expr.KVPairs, error) {
	// Parse { token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.LBRACKET {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"{"}, pos)
	}

	var pairs expr.KVPairs
	pairs.SelfReferenced = true
	var pair expr.KVPair
	var err error

	fields := make(map[string]struct{})

	// Parse kv pairs.
	for {
		if pair, err = p.parseKV(); err != nil {
			p.Unscan()
			break
		}

		if _, ok := fields[pair.K]; ok {
			return nil, stringutil.Errorf("duplicate field %q", pair.K)
		}
		fields[pair.K] = struct{}{}

		pairs.Pairs = append(pairs.Pairs, pair)

		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.COMMA {
			p.Unscan()
			break
		}
	}

	// Parse required } token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.RBRACKET {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"}"}, pos)
	}

	return &pairs, nil
}

// parseKV parses a key-value pair in the form IDENT : Expr.
func (p *Parser) parseKV() (expr.KVPair, error) {
	var k string

	tok, pos, lit := p.ScanIgnoreWhitespace()
	if tok == scanner.IDENT || tok == scanner.STRING {
		k = lit
	} else {
		return expr.KVPair{}, newParseError(scanner.Tokstr(tok, lit), []string{"ident", "string"}, pos)
	}

	tok, pos, lit = p.ScanIgnoreWhitespace()
	if tok != scanner.COLON {
		p.Unscan()
		return expr.KVPair{}, newParseError(scanner.Tokstr(tok, lit), []string{":"}, pos)
	}

	e, _, err := p.ParseExpr()
	if err != nil {
		return expr.KVPair{}, err
	}

	return expr.KVPair{
		K: k,
		V: e,
	}, nil
}

// parsePath parses a path to a specific value.
func (p *Parser) parsePath() (document.Path, error) {
	var path document.Path
	// parse first mandatory ident
	chunk, err := p.parseIdent()
	if err != nil {
		return nil, err
	}
	path = append(path, document.PathFragment{
		FieldName: chunk,
	})

LOOP:
	for {
		// scan the very next token.
		// if can be either a '.' or a '['
		// Otherwise, unscan and return the path
		tok, _, _ := p.Scan()
		switch tok {
		case scanner.DOT:
			// scan the next token for an ident
			tok, pos, lit := p.Scan()
			if tok != scanner.IDENT {
				return nil, newParseError(lit, []string{"identifier"}, pos)
			}
			path = append(path, document.PathFragment{
				FieldName: lit,
			})
		case scanner.LSBRACKET:
			// scan the next token for an integer
			tok, pos, lit := p.Scan()
			if tok != scanner.INTEGER || lit[0] == '-' {
				return nil, newParseError(lit, []string{"array index"}, pos)
			}
			idx, err := strconv.Atoi(lit)
			if err != nil {
				return nil, newParseError(lit, []string{"integer"}, pos)
			}
			path = append(path, document.PathFragment{
				ArrayIndex: idx,
			})
			// scan the next token for a closing left bracket
			tok, pos, lit = p.Scan()
			if tok != scanner.RSBRACKET {
				return nil, newParseError(lit, []string{"]"}, pos)
			}
		default:
			p.Unscan()
			break LOOP
		}
	}

	return path, nil
}

func (p *Parser) parseExprListUntil(rightToken scanner.Token) (expr.LiteralExprList, error) {
	var exprList expr.LiteralExprList
	var expr expr.Expr
	var err error

	// Parse expressions.
	for {
		if expr, _, err = p.ParseExpr(); err != nil {
			p.Unscan()
			break
		}

		exprList = append(exprList, expr)

		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.COMMA {
			p.Unscan()
			break
		}
	}

	// Parse required ) or ] token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != rightToken {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{rightToken.String()}, pos)
	}

	return exprList, nil
}

func (p *Parser) parseExprList(leftToken, rightToken scanner.Token) (expr.LiteralExprList, error) {
	// Parse ( or [ token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != leftToken {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{leftToken.String()}, pos)
	}

	return p.parseExprListUntil(rightToken)
}

// parseFunction parses a function call.
// a function is an identifier followed by a parenthesis,
// an optional coma-separated list of expressions and a closing parenthesis.
func (p *Parser) parseFunction() (expr.Expr, error) {
	// Parse function name.
	fname, err := p.parseIdent()
	if err != nil {
		return nil, err
	}

	// Parse required ( token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.LPAREN {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"("}, pos)
	}

	// Special case: If the function is COUNT, support the special case COUNT(*)
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok == scanner.MUL {
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.RPAREN {
			return nil, newParseError(scanner.Tokstr(tok, lit), []string{")"}, pos)
		}

		return &expr.CountFunc{Wildcard: true}, nil
	}
	p.Unscan()

	// Check if the function is called without arguments.
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == scanner.RPAREN {
		return p.functions.GetFunc(fname)
	}
	p.Unscan()

	var exprs []expr.Expr

	// Parse expressions.
	for {
		e, _, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, e)

		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != scanner.COMMA {
			p.Unscan()
			break
		}
	}

	// Parse required ) token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.RPAREN {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{")"}, pos)
	}

	return p.functions.GetFunc(fname, exprs...)
}

// parseCastExpression parses a string of the form CAST(expr AS type).
func (p *Parser) parseCastExpression() (expr.Expr, error) {
	// Parse required CAST token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.CAST {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"CAST"}, pos)
	}

	// Parse required ( token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.LPAREN {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"("}, pos)
	}

	// parse required expression.
	e, _, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}

	// Parse required AS token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.AS {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"AS"}, pos)
	}

	// Parse required typename.
	tp, err := p.parseType()
	if err != nil {
		return nil, err
	}

	// Parse required ) token.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.RPAREN {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{")"}, pos)
	}

	return expr.CastFunc{Expr: e, CastAs: tp}, nil
}
