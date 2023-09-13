package store

import (
	"berith-swap/bridge/store/mariadb"
	"context"
	"database/sql"
	"fmt"
)

const (
	DBDriver = "mysql"
)

type Store struct {
	mariadb.Queries
	db *sql.DB
}

func NewStore(source string) (*Store, error) {
	conn, err := sql.Open("mysql", source)
	if err != nil {
		return nil, err
	}

	return &Store{
		Queries: *mariadb.New(conn),
		db:      conn,
	}, nil
}

func (store *Store) execTx(ctx context.Context, fn func(*mariadb.Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := mariadb.New(tx)
	err = fn(q)

	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err:%w, rb err:%w", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
