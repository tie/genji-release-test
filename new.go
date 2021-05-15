// +build !wasm

package genji

import (
	"context"

	"github.com/tie/genji-release-test/database"
	"github.com/tie/genji-release-test/document/encoding/msgpack"
	"github.com/tie/genji-release-test/engine"
)

// New initializes the DB using the given engine.
func New(ctx context.Context, ng engine.Engine) (*DB, error) {
	db, err := database.New(ctx, ng, database.Options{Codec: msgpack.NewCodec()})
	if err != nil {
		return nil, err
	}

	return &DB{
		DB:  db,
		ctx: context.Background(),
	}, nil
}
