package db

import (
	"github.com/jmoiron/sqlx"
)

// TxExecer performs the task exec in a transaction against db.
type TxExecer struct {
	db   *sqlx.DB
	exec func(tx *sqlx.Tx) error
}

// NewTxExecer creates a TxExecer from db and f.
func NewTxExecer(db *sqlx.DB, f func(tx *sqlx.Tx) error) *TxExecer {
	return &TxExecer{
		db:   db,
		exec: f,
	}
}

// Exec creates a transaction on db and runs the task.
// It takes care of BEGIN/COMMIT/ROLLBACK and will rollback if
// an error occurs or is returned after invoking exec.
func (e *TxExecer) Exec() error {
	tx, err := e.db.Beginx()
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
