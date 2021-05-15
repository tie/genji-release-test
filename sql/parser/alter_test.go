package parser_test

import (
	"testing"

	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/query"
	"github.com/tie/genji-release-test/sql/parser"
	"github.com/tie/genji-release-test/testutil"
	"github.com/stretchr/testify/require"
)

func TestParserAlterTable(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected query.Statement
		errored  bool
	}{
		{"Basic", "ALTER TABLE foo RENAME TO bar", query.AlterStmt{TableName: "foo", NewTableName: "bar"}, false},
		{"With error / missing TABLE keyword", "ALTER foo RENAME TO bar", query.AlterStmt{}, true},
		{"With error / two identifiers for table name", "ALTER TABLE foo baz RENAME TO bar", query.AlterStmt{}, true},
		{"With error / two identifiers for new table name", "ALTER TABLE foo RENAME TO bar baz", query.AlterStmt{}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := parser.ParseQuery(test.s)
			if test.errored {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, q.Statements, 1)
			require.EqualValues(t, test.expected, q.Statements[0])
		})
	}
}

func TestParserAlterTableAddField(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected query.Statement
		errored  bool
	}{
		{"Basic", "ALTER TABLE foo ADD FIELD bar", query.AlterTableAddField{TableName: "foo",
			Constraint: database.FieldConstraint{},
		}, true},
		{"With type", "ALTER TABLE foo ADD FIELD bar integer", query.AlterTableAddField{TableName: "foo",
			Constraint: database.FieldConstraint{
				Path: document.Path(testutil.ParsePath(t, "bar")),
				Type: document.IntegerValue,
			},
		}, false},
		{"With not null", "ALTER TABLE foo ADD FIELD bar NOT NULL", query.AlterTableAddField{TableName: "foo",
			Constraint: database.FieldConstraint{
				Path:      document.Path(testutil.ParsePath(t, "bar")),
				IsNotNull: true,
			},
		}, false},
		{"With primary key", "ALTER TABLE foo ADD FIELD bar PRIMARY KEY", query.AlterTableAddField{}, true},
		{"With multiple constraints", "ALTER TABLE foo ADD FIELD bar integer NOT NULL DEFAULT 0", query.AlterTableAddField{TableName: "foo",
			Constraint: database.FieldConstraint{
				Path:         document.Path(testutil.ParsePath(t, "bar")),
				Type:         document.IntegerValue,
				IsNotNull:    true,
				DefaultValue: document.NewIntegerValue(0),
			},
		}, false},
		{"With error / missing FIELD keyword", "ALTER TABLE foo ADD bar", nil, true},
		{"With error / missing field name", "ALTER TABLE foo ADD FIELD", nil, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := parser.ParseQuery(test.s)
			if test.errored {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, q.Statements, 1)
			require.EqualValues(t, test.expected, q.Statements[0])
		})
	}
}
