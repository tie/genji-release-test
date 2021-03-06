package scanner_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tie/genji-release-test/sql/scanner"
)

// Ensure the scanner can scan tokens correctly.
func TestScanner_Scan(t *testing.T) {
	var tests = []struct {
		s   string
		tok scanner.Token
		lit string
		pos scanner.Pos
		raw string
	}{
		// Special tokens (EOF, ILLEGAL, WS)
		{s: ``, tok: scanner.EOF, raw: ``},
		{s: `#`, tok: scanner.ILLEGAL, lit: `#`, raw: `#`},
		{s: ` `, tok: scanner.WS, lit: " ", raw: ` `},
		{s: "\t", tok: scanner.WS, lit: "\t", raw: "\t"},
		{s: "\n", tok: scanner.WS, lit: "\n", raw: "\n"},
		{s: "\r", tok: scanner.WS, lit: "\n", raw: "\n"},
		{s: "\r\n", tok: scanner.WS, lit: "\n", raw: "\n"},
		{s: "\rX", tok: scanner.WS, lit: "\n", raw: "\n"},
		{s: "\n\r", tok: scanner.WS, lit: "\n\n", raw: "\n\n"},
		{s: " \n\t \r\n\t", tok: scanner.WS, lit: " \n\t \n\t", raw: " \n\t \n\t"},
		{s: " foo", tok: scanner.WS, lit: " ", raw: " "},

		// Numeric operators
		{s: `+`, tok: scanner.ADD, raw: `+`},
		{s: `-`, tok: scanner.SUB, raw: `-`},
		{s: `*`, tok: scanner.MUL, raw: `*`},
		{s: `/`, tok: scanner.DIV, raw: `/`},
		{s: `%`, tok: scanner.MOD, raw: `%`},

		// Logical operators
		{s: `AND`, tok: scanner.AND, raw: `AND`},
		{s: `and`, tok: scanner.AND, raw: `and`},
		{s: `OR`, tok: scanner.OR, raw: `OR`},
		{s: `or`, tok: scanner.OR, raw: `or`},

		// Comparison operators
		{s: `=`, tok: scanner.EQ, raw: `=`},
		{s: `==`, tok: scanner.EQ, raw: `==`},
		{s: `<>`, tok: scanner.NEQ, raw: `<>`},
		{s: `! `, tok: scanner.ILLEGAL, lit: "!", raw: `!`},
		{s: `<`, tok: scanner.LT, raw: `<`},
		{s: `<=`, tok: scanner.LTE, raw: `<=`},
		{s: `>`, tok: scanner.GT, raw: `>`},
		{s: `>=`, tok: scanner.GTE, raw: `>=`},
		{s: `IN`, tok: scanner.IN, raw: `IN`},
		{s: `IS`, tok: scanner.IS, raw: `IS`},
		{s: `LIKE`, tok: scanner.LIKE, raw: `LIKE`},
		{s: `||`, tok: scanner.CONCAT, raw: `||`},

		// Misc tokens
		{s: `(`, tok: scanner.LPAREN, raw: `(`},
		{s: `)`, tok: scanner.RPAREN, raw: `)`},
		{s: `{`, tok: scanner.LBRACKET, raw: `{`},
		{s: `}`, tok: scanner.RBRACKET, raw: `}`},
		{s: `[`, tok: scanner.LSBRACKET, raw: `[`},
		{s: `]`, tok: scanner.RSBRACKET, raw: `]`},
		{s: `,`, tok: scanner.COMMA, raw: `,`},
		{s: `;`, tok: scanner.SEMICOLON, raw: `;`},
		{s: `.`, tok: scanner.DOT, raw: `.`},
		{s: `=~`, tok: scanner.EQREGEX, raw: `=~`},
		{s: `!~`, tok: scanner.NEQREGEX, raw: `!~`},
		{s: `:`, tok: scanner.COLON, raw: `:`},
		{s: `::`, tok: scanner.DOUBLECOLON, raw: `::`},
		{s: `--`, tok: scanner.COMMENT, raw: `--`},
		{s: `--10.3`, tok: scanner.COMMENT, lit: ``, raw: `--10.3`},

		// Identifiers
		{s: `foo`, tok: scanner.IDENT, lit: `foo`, raw: `foo`},
		{s: `_foo`, tok: scanner.IDENT, lit: `_foo`, raw: `_foo`},
		{s: `Zx12_3U_-`, tok: scanner.IDENT, lit: `Zx12_3U_`, raw: `Zx12_3U_`},
		{s: "`foo`", tok: scanner.IDENT, lit: "foo", raw: "`foo`"},
		{s: "`foo\bar`", tok: scanner.IDENT, lit: "foo\bar", raw: "`foo\bar`"},
		{s: "`foo\\bar`", tok: scanner.BADESCAPE, lit: `\b`, pos: scanner.Pos{Line: 0, Char: 5}, raw: "`foo\\b"},
		{s: "`foo\\`bar\\``", tok: scanner.IDENT, lit: "foo`bar`", raw: "`foo\\`bar\\``"},
		{s: "test`", tok: scanner.BADSTRING, lit: "", pos: scanner.Pos{Line: 0, Char: 3}, raw: "test`"},
		{s: "`test", tok: scanner.BADSTRING, lit: "test", raw: "`test"},
		{s: "$host", tok: scanner.NAMEDPARAM, lit: "$host", raw: "$host"},
		{s: "$`host param`", tok: scanner.NAMEDPARAM, lit: "$host param", raw: "$`host param`"},
		{s: "?", tok: scanner.POSITIONALPARAM, lit: "", raw: "?"},

		// Booleans
		{s: `true`, tok: scanner.TRUE, raw: `true`},
		{s: `false`, tok: scanner.FALSE, raw: `false`},

		// Null
		{s: `null`, tok: scanner.NULL, raw: `null`},
		{s: `NULL`, tok: scanner.NULL, raw: `NULL`},

		// Strings
		{s: `'testing 123!'`, tok: scanner.STRING, lit: `testing 123!`, raw: `'testing 123!'`},
		{s: `'foo\nbar'`, tok: scanner.STRING, lit: "foo\nbar", raw: `'foo\nbar'`},
		{s: `'foo\\bar'`, tok: scanner.STRING, lit: "foo\\bar", raw: `'foo\\bar'`},
		{s: `'test`, tok: scanner.BADSTRING, lit: `test`, raw: `'test`},
		{s: "'test\nfoo", tok: scanner.BADSTRING, lit: `test`, raw: "'test\n"},
		{s: `'test\g'`, tok: scanner.BADESCAPE, lit: `\g`, pos: scanner.Pos{Line: 0, Char: 6}, raw: `'test\g`},
		{s: `"testing 123!"`, tok: scanner.STRING, lit: `testing 123!`, raw: `"testing 123!"`},
		{s: `"foo\nbar"`, tok: scanner.STRING, lit: "foo\nbar", raw: `"foo\nbar"`},
		{s: `"foo\\bar"`, tok: scanner.STRING, lit: "foo\\bar", raw: `"foo\\bar"`},
		{s: `"test`, tok: scanner.BADSTRING, lit: `test`, raw: `"test`},
		{s: "\"test\nfoo", tok: scanner.BADSTRING, lit: `test`, raw: "\"test\n"},
		{s: `"test\g"`, tok: scanner.BADESCAPE, lit: `\g`, pos: scanner.Pos{Line: 0, Char: 6}, raw: `"test\g`},

		// Numbers
		{s: `100`, tok: scanner.INTEGER, lit: `100`, raw: `100`},
		{s: `100.23`, tok: scanner.NUMBER, lit: `100.23`, raw: `100.23`},
		{s: `.23`, tok: scanner.NUMBER, lit: `.23`, raw: `.23`},
		{s: `10.3s`, tok: scanner.NUMBER, lit: `10.3`, raw: `10.3`},
		{s: `-10.3`, tok: scanner.NUMBER, lit: `-10.3`, raw: `-10.3`},

		// Keywords
		{s: `ADD`, tok: scanner.ADD_KEYWORD, raw: `ADD`},
		{s: `ALTER`, tok: scanner.ALTER, raw: `ALTER`},
		{s: `AS`, tok: scanner.AS, raw: `AS`},
		{s: `ASC`, tok: scanner.ASC, raw: `ASC`},
		{s: `BY`, tok: scanner.BY, raw: `BY`},
		{s: `BEGIN`, tok: scanner.BEGIN, raw: `BEGIN`},
		{s: `BETWEEN`, tok: scanner.BETWEEN, raw: `BETWEEN`},
		{s: `CAST`, tok: scanner.CAST, raw: `CAST`},
		{s: `COMMIT`, tok: scanner.COMMIT, raw: `COMMIT`},
		{s: `CREATE`, tok: scanner.CREATE, raw: `CREATE`},
		{s: `EXPLAIN`, tok: scanner.EXPLAIN, raw: `EXPLAIN`},
		{s: `DEFAULT`, tok: scanner.DEFAULT, raw: `DEFAULT`},
		{s: `DELETE`, tok: scanner.DELETE, raw: `DELETE`},
		{s: `DESC`, tok: scanner.DESC, raw: `DESC`},
		{s: `DISTINCT`, tok: scanner.DISTINCT, raw: `DISTINCT`},
		{s: `DROP`, tok: scanner.DROP, raw: `DROP`},
		{s: `FIELD`, tok: scanner.FIELD, raw: `FIELD`},
		{s: `FROM`, tok: scanner.FROM, raw: `FROM`},
		{s: `GROUP`, tok: scanner.GROUP, raw: `GROUP`},
		{s: `INSERT`, tok: scanner.INSERT, raw: `INSERT`},
		{s: `INTO`, tok: scanner.INTO, raw: `INTO`},
		{s: `LIMIT`, tok: scanner.LIMIT, raw: `LIMIT`},
		{s: `ONLY`, tok: scanner.ONLY, raw: `ONLY`},
		{s: `OFFSET`, tok: scanner.OFFSET, raw: `OFFSET`},
		{s: `ORDER`, tok: scanner.ORDER, raw: `ORDER`},
		{s: `PRIMARY`, tok: scanner.PRIMARY, raw: `PRIMARY`},
		{s: `READ`, tok: scanner.READ, raw: `READ`},
		{s: `REINDEX`, tok: scanner.REINDEX, raw: `REINDEX`},
		{s: `RENAME`, tok: scanner.RENAME, raw: `RENAME`},
		{s: `RETURNING`, tok: scanner.RETURNING, raw: `RETURNING`},
		{s: `ROLLBACK`, tok: scanner.ROLLBACK, raw: `ROLLBACK`},
		{s: `SELECT`, tok: scanner.SELECT, raw: `SELECT`},
		{s: `SET`, tok: scanner.SET, raw: `SET`},
		{s: `TABLE`, tok: scanner.TABLE, raw: `TABLE`},
		{s: `TO`, tok: scanner.TO, raw: `TO`},
		{s: `TRANSACTION`, tok: scanner.TRANSACTION, raw: `TRANSACTION`},
		{s: `UPDATE`, tok: scanner.UPDATE, raw: `UPDATE`},
		{s: `UNSET`, tok: scanner.UNSET, raw: `UNSET`},
		{s: `VALUES`, tok: scanner.VALUES, raw: `VALUES`},
		{s: `WHERE`, tok: scanner.WHERE, raw: `WHERE`},
		{s: `WRITE`, tok: scanner.WRITE, raw: `WRITE`},
		{s: `seLECT`, tok: scanner.SELECT, raw: `seLECT`}, // case insensitive

		// types
		{s: "BYTES", tok: scanner.TYPEBYTES, raw: `BYTES`},
		{s: "BOOL", tok: scanner.TYPEBOOL, raw: `BOOL`},
		{s: "DOUBLE", tok: scanner.TYPEDOUBLE, raw: `DOUBLE`},
		{s: "INTEGER", tok: scanner.TYPEINTEGER, raw: `INTEGER`},
		{s: "TEXT", tok: scanner.TYPETEXT, raw: `TEXT`},
	}

	for i, tt := range tests {
		s := scanner.NewScanner(strings.NewReader(tt.s))
		ti := s.Scan()
		if tt.tok != ti.Tok {
			t.Errorf("%d. %q token mismatch: exp=%q got=%q <%q>", i, tt.s, tt.tok, ti.Tok, ti.Lit)
		} else if tt.pos.Line != ti.Pos.Line || tt.pos.Char != ti.Pos.Char {
			t.Errorf("%d. %q pos mismatch: exp=%#v got=%#v", i, tt.s, tt.pos, ti.Pos)
		} else if tt.lit != ti.Lit {
			t.Errorf("%d. %q literal mismatch: exp=%q got=%q", i, tt.s, tt.lit, ti.Lit)
		} else if tt.raw != ti.Raw {
			t.Errorf("%d. %q raw mismatch: exp=%q got=%q", i, tt.s, tt.raw, ti.Raw)
		}
	}
}

// Ensure the scanner can scan a series of tokens correctly.
func TestScanner_Scan_Multi(t *testing.T) {
	exp := []scanner.TokenInfo{
		{Tok: scanner.SELECT, Pos: scanner.Pos{Line: 0, Char: 0}, Lit: "", Raw: "SELECT"},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 6}, Lit: " ", Raw: " "},
		{Tok: scanner.IDENT, Pos: scanner.Pos{Line: 0, Char: 7}, Lit: "value", Raw: "value"},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 12}, Lit: " ", Raw: " "},
		{Tok: scanner.FROM, Pos: scanner.Pos{Line: 0, Char: 13}, Lit: "", Raw: "from"},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 17}, Lit: " ", Raw: " "},
		{Tok: scanner.IDENT, Pos: scanner.Pos{Line: 0, Char: 18}, Lit: "my_table", Raw: "my_table"},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 26}, Lit: " ", Raw: " "},
		{Tok: scanner.WHERE, Pos: scanner.Pos{Line: 0, Char: 27}, Lit: "", Raw: "WHERE"},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 32}, Lit: " ", Raw: " "},
		{Tok: scanner.IDENT, Pos: scanner.Pos{Line: 0, Char: 33}, Lit: "a", Raw: "a"},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 34}, Lit: " ", Raw: " "},
		{Tok: scanner.EQ, Pos: scanner.Pos{Line: 0, Char: 35}, Lit: "", Raw: "="},
		{Tok: scanner.WS, Pos: scanner.Pos{Line: 0, Char: 36}, Lit: " ", Raw: " "},
		{Tok: scanner.STRING, Pos: scanner.Pos{Line: 0, Char: 36}, Lit: "b", Raw: "'b'"},
		{Tok: scanner.EOF, Pos: scanner.Pos{Line: 0, Char: 40}, Lit: "", Raw: ""},
	}

	// Create a scanner.
	v := `SELECT value from my_table WHERE a = 'b'`
	s := scanner.NewScanner(strings.NewReader(v))

	// Continually scan until we reach the end.
	var act []scanner.TokenInfo
	for {
		ti := s.Scan()
		act = append(act, ti)
		if ti.Tok == scanner.EOF {
			break
		}
	}

	// Verify the token counts match.
	if len(exp) != len(act) {
		t.Fatalf("token count mismatch: exp=%d, got=%d", len(exp), len(act))
	}

	// Verify each token matches.
	for i := range exp {
		if !reflect.DeepEqual(exp[i], act[i]) {
			t.Fatalf("%d. token mismatch:\n\nexp=%#v\n\ngot=%#v", i, exp[i], act[i])
		}
	}
}

// Ensure the library can correctly scan strings.
func TestScanString(t *testing.T) {
	var tests = []struct {
		in  string
		out string
		err string
	}{
		{in: `""`, out: ``},
		{in: `"foo bar"`, out: `foo bar`},
		{in: `'foo bar'`, out: `foo bar`},
		{in: `"foo\nbar"`, out: "foo\nbar"},
		{in: `"foo\\bar"`, out: `foo\bar`},
		{in: `"foo\"bar"`, out: `foo"bar`},
		{in: `'foo\'bar'`, out: `foo'bar`},

		{in: `"foo` + "\n", out: `foo`, err: "bad string"}, // newline in string
		{in: `"foo`, out: `foo`, err: "bad string"},        // unclosed quotes
		{in: `"foo\xbar"`, out: `\x`, err: "bad escape"},   // invalid escape
	}

	for i, tt := range tests {
		out, err := scanner.ScanString(strings.NewReader(tt.in))
		if tt.err != errstring(err) {
			t.Errorf("%d. %s: error: exp=%s, got=%s", i, tt.in, tt.err, err)
		} else if tt.out != out {
			t.Errorf("%d. %s: out: exp=%s, got=%s", i, tt.in, tt.out, out)
		}
	}
}

// errstring converts an error to its string representation.
func errstring(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// Test scanning regex
func TestScanRegex(t *testing.T) {
	var tests = []struct {
		in  string
		tok scanner.Token
		lit string
		err string
	}{
		{in: `/^payments\./`, tok: scanner.REGEX, lit: `^payments\.`},
		{in: `/foo\/bar/`, tok: scanner.REGEX, lit: `foo/bar`},
		{in: `/foo\\/bar/`, tok: scanner.REGEX, lit: `foo\/bar`},
		{in: `/foo\\bar/`, tok: scanner.REGEX, lit: `foo\\bar`},
		{in: `/http\:\/\/www\.example\.com/`, tok: scanner.REGEX, lit: `http\://www\.example\.com`},
	}

	for i, tt := range tests {
		s := scanner.NewScanner(strings.NewReader(tt.in))
		ti := s.ScanRegex()
		if ti.Tok != tt.tok {
			t.Errorf("%d. %s: error:\n\texp=%s\n\tgot=%s\n", i, tt.in, tt.tok.String(), ti.Tok.String())
		}
		if ti.Lit != tt.lit {
			t.Errorf("%d. %s: error:\n\texp=%s\n\tgot=%s\n", i, tt.in, tt.lit, ti.Lit)
		}
	}
}
