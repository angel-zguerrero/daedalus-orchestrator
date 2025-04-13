package db

import (
	"fmt"

	"github.com/linxGnu/grocksdb"
)

func openDB(dbPath string) (*grocksdb.DB, error) {
	fmt.Println("🗄️  Opening db:", dbPath)
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetInfoLogLevel(grocksdb.WarnInfoLogLevel)
	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	return db, nil
}
