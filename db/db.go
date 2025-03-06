package db

import (
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

func OpenDB(dbPath string) (*badger.DB, error) {
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // 禁用默认日志
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	return db, nil
}
