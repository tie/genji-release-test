package dbutil

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/tie/genji-release-test"
	"github.com/tie/genji-release-test/engine"
	"github.com/tie/genji-release-test/engine/badgerengine"
	"github.com/tie/genji-release-test/engine/boltengine"
	"github.com/tie/genji-release-test/engine/memoryengine"
	"github.com/tie/genji-release-test/stringutil"
	"go.etcd.io/bbolt"
)

// OpenDB opens a database at the given path, using the selected engine.
func OpenDB(ctx context.Context, dbPath, engineName string) (*genji.DB, error) {
	var (
		ng  engine.Engine
		err error
	)

	switch engineName {
	case "memory":
		ng = memoryengine.NewEngine()
	case "bolt":
		ng, err = boltengine.NewEngine(dbPath, 0660, &bbolt.Options{
			Timeout: 100 * time.Millisecond,
		})
		if err == bbolt.ErrTimeout {
			return nil, errors.New("database is locked")
		}
	case "badger":
		ng, err = badgerengine.NewEngine(badger.DefaultOptions(dbPath).WithLogger(nil))
		if err != nil && strings.HasPrefix(err.Error(), "Cannot acquire directory lock") {
			return nil, errors.New("database is locked")
		}
	default:
		return nil, stringutil.Errorf(`engine should be "bolt" or "badger", got %q`, engineName)
	}
	if err != nil {
		return nil, err
	}

	return genji.New(ctx, ng)
}
