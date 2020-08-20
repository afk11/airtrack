package db

import (
	"database/sql"
)

type TxExecer struct {
	db   *sql.DB
	exec func(tx *sql.Tx) error
}

func NewTxExecer(db *sql.DB, f func(tx *sql.Tx) error) *TxExecer {
	return &TxExecer{
		db:   db,
		exec: f,
	}
}
func (e *TxExecer) Exec() error {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	f := e.exec
	err = f(tx)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			panic(rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			panic(rollbackErr)
		}
		return err
	}

	return nil
}
