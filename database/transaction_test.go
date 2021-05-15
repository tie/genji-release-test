package database_test

import (
	"context"
	"testing"

	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document/encoding/msgpack"
	"github.com/tie/genji-release-test/engine/memoryengine"
	"github.com/stretchr/testify/require"
)

func newTestDB(t testing.TB) (*database.Database, func()) {
	db, err := database.New(context.Background(), memoryengine.NewEngine(), database.Options{
		Codec: msgpack.NewCodec(),
	})
	require.NoError(t, err)

	return db, func() {
		db.Close()
	}
}

func newTestTx(t testing.TB) (*database.Database, *database.Transaction, func()) {
	db, cleanup := newTestDB(t)

	tx, err := db.Begin(true)
	require.NoError(t, err)

	return db, tx, func() {
		tx.Rollback()
		cleanup()
	}
}
