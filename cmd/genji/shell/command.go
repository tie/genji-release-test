package shell

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tie/genji-release-test"
	"github.com/tie/genji-release-test/cmd/genji/dbutil"
	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document"
	"github.com/tie/genji-release-test/engine"
	"github.com/tie/genji-release-test/stringutil"
)

type command struct {
	Name        string
	Options     string
	DisplayName string
	Description string
	Aliases     []string
}

func (c *command) Usage() string {
	return fmt.Sprintf("%s %s", c.DisplayName, c.Options)
}

var commands = []command{
	{
		Name:        ".exit",
		DisplayName: ".exit or exit",
		Description: "Exit this program.",
		Aliases:     []string{"exit"},
	},
	{
		Name:        ".help",
		DisplayName: ".help or help",
		Description: "List all commands.",
		Aliases:     []string{"help"},
	},
	{
		Name:        ".tables",
		DisplayName: ".tables",
		Description: "List names of tables.",
	},
	{
		Name:        ".indexes",
		Options:     "[table_name]",
		DisplayName: ".indexes",
		Description: "Display all indexes or the indexes of the given table name.",
	},
	{
		Name:        ".dump",
		Options:     "[table_name]",
		DisplayName: ".dump",
		Description: "Dump database content or table content as SQL statements.",
	},
	{
		Name:        ".save",
		Options:     "[badger] [filename]",
		DisplayName: ".save",
		Description: "Save database content in the specified file.",
	},
	{
		Name:        ".schema",
		Options:     "[table_name]",
		DisplayName: ".schema",
		Description: "Show the CREATE statements of all tables or of the selected ones.",
	},
	{
		Name:        ".import",
		Options:     "TYPE FILE table",
		DisplayName: ".import",
		Description: "Import data from a file. Only supported type is 'csv'",
	},
}

func getUsage(cmdName string) string {
	for _, c := range commands {
		if c.Name == cmdName {
			return c.Usage()
		}
	}

	return ""
}

// runHelpCmd shows all available commands.
func runHelpCmd() error {
	for _, c := range commands {
		// indentation for readability.
		spaces := 25
		indent := spaces - len(c.DisplayName) - len(c.Options)
		fmt.Printf("%s %s %*s %s\n", c.DisplayName, c.Options, indent, "", c.Description)
	}

	return nil
}

// runTablesCmd displays all tables.
func runTablesCmd(db *genji.DB, w io.Writer) error {
	res, err := db.Query("SELECT table_name FROM __genji_tables")
	if err != nil {
		return err
	}
	defer res.Close()

	return res.Iterate(func(d document.Document) error {
		var tableName string
		err = document.Scan(d, &tableName)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, tableName)
		return err
	})
}

// runIndexesCmd displays a list of indexes. If table is non-empty, it only
// display that table's indexes. If not, it displays all indexes.
func runIndexesCmd(db *genji.DB, tableName string, w io.Writer) error {
	return db.View(func(tx *genji.Tx) error {
		q := "SELECT * FROM __genji_indexes"

		if tableName != "" {
			// ensure table exists
			_, err := tx.QueryDocument("SELECT 1 FROM __genji_tables WHERE table_name = ? LIMIT 1", tableName)
			if err != nil {
				if err == database.ErrDocumentNotFound {
					return stringutil.Errorf("%w: %q", database.ErrTableNotFound, tableName)
				}
				return err
			}

			q += " WHERE table_name = ?"
		}

		res, err := tx.Query(q, tableName)
		if err != nil {
			return err
		}
		defer res.Close()

		return res.Iterate(func(d document.Document) error {
			var index database.IndexInfo

			if err := index.ScanDocument(d); err != nil {
				return err
			}

			var paths []string
			for _, path := range index.Paths {
				paths = append(paths, path.String())
			}

			fmt.Fprintf(w, "%s ON %s (%s)\n", index.IndexName, index.TableName, strings.Join(paths, ", "))

			return nil
		})
	})
}

// runSaveCommand saves the currently opened database at the given path.
// If a path already exists, existing values in the target database will be overwritten.
func runSaveCmd(ctx context.Context, db *genji.DB, engineName string, dbPath string) error {
	tx, err := db.Begin(false)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Open the new database
	otherDB, err := dbutil.OpenDB(ctx, dbPath, engineName)
	if err != nil {
		return err
	}
	otherDB = otherDB.WithContext(ctx)
	defer otherDB.Close()

	otherTx, err := otherDB.Begin(true)
	if err != nil {
		return err
	}
	defer otherTx.Rollback()

	// Find all tables
	tables, err := tx.Query("SELECT table_name FROM __genji_tables")
	if err != nil {
		return err
	}
	defer tables.Close()

	err = tables.Iterate(func(d document.Document) error {
		// Get table name.
		var tableName string
		if err := document.Scan(d, &tableName); err != nil {
			return err
		}

		table, err := tx.GetTable(tableName)
		if err != nil {
			return err
		}

		ti := table.Info()

		err = otherTx.CreateTable(tableName, ti)
		if err != nil {
			return err
		}
		otherTable, err := otherTx.GetTable(tableName)
		if err != nil {
			return err
		}

		it := table.Store.Iterator(engine.IteratorOptions{})
		defer it.Close()

		var b []byte
		for it.Seek(nil); it.Valid(); it.Next() {
			itm := it.Item()
			b, err := itm.ValueCopy(b)
			if err != nil {
				return err
			}

			err = otherTable.Store.Put(itm.Key(), b)
			if err != nil {
				return err
			}
		}

		return err
	})
	if err != nil {
		return err
	}

	// Find all indexes
	indexes, err := tx.Query("SELECT * FROM __genji_indexes")
	if err != nil {
		return err
	}
	defer indexes.Close()

	err = indexes.Iterate(func(d document.Document) error {
		var index database.IndexInfo

		if err := index.ScanDocument(d); err != nil {
			return err
		}

		err = otherTx.CreateIndex(&index)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = otherTx.ReIndexAll()
	if err != nil {
		return err
	}

	err = otherTx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func runImportCmd(ctx context.Context, db *genji.DB, fileType, path, table string) error {
	if strings.ToLower(fileType) != "csv" {
		return errors.New("TYPE should be csv")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r := csv.NewReader(f)

	err = tx.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s", table))
	if err != nil {
		return err
	}

	headers, err := r.Read()
	if err != nil {
		return err
	}

	for {
		columns, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = tx.Exec("INSERT INTO "+table+" VALUES ?", document.NewFromCSV(headers, columns))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
