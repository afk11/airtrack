package tracker

import (
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/test"
	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"os"
	"sync"
	"time"
)

type TestDB struct {
	conf  *test.TestDbConfig
	tz    *time.Location
	db    string
	inUse bool
}

var dbs []*TestDB

var claimNextDBLock = &sync.Mutex{}

func init() {
	tz := test.MustLoadTestTimeZone()

	dbs = make([]*TestDB, 4)
	for i := 0; i < 4; i++ {
		dbs[i] = &TestDB{
			db:    fmt.Sprintf("airtrack_%d", i),
			tz:    tz.Tz,
			inUse: false}
	}
}

func claimNextDB() *TestDB {
	claimNextDBLock.Lock()
	defer claimNextDBLock.Unlock()

	for _, testDB := range dbs {
		if !testDB.inUse {
			testDB.inUse = true
			return testDB
		}
	}

	panic("Failed to claim a DB")

	return nil
}

func dropTables(db *sqlx.DB) error {
	rows, err := db.Query(`SHOW TABLES`)
	if err != nil {
		return errors.Wrapf(err, "querying for tables")
	}

	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			return err
		}
		tables = append(tables, table)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	for _, table := range tables {
		_, err = tx.Exec("DROP TABLE " + table)
		if err != nil {
			return errors.Wrapf(err, "drop table")
		}
	}
	err = tx.Commit()
	if err != nil {
		rerr := tx.Rollback()
		if rerr != nil {
			return rerr
		}
		return err
	}
	return nil
}

func initDB() (*sqlx.DB, goqu.DialectWrapper, string, func()) {
	testDB := claimNextDB()

	tmpFile := "/tmp/"+testDB.db+".sqlite3"
	if _, err := os.Stat(tmpFile); err == nil {
		err = os.Remove(tmpFile)
		if err != nil {
			panic(err)
		}
	}

	dbUrl := fmt.Sprintf("file:"+tmpFile)
	sqlConn, err := sql.Open("sqlite3", dbUrl)
	dbConn := sqlx.NewDb(sqlConn, "sqlite3")
	if err != nil {
		panic(fmt.Sprintf("error creating database connection %s", err.Error()))
	}

	migrate, err := test.InitSqliteMigration(testDB.db, tmpFile, dbConn.DB)
	if err != nil {
		panic(err)
	}
	err = migrate.Up()
	if err != nil {
		panic(err)
	}

	// create close function
	closeFn := func() {
		dbConn.Close()
		testDB.inUse = false
	}
	dialect := goqu.Dialect("sqlite3")
	return dbConn, dialect, testDB.db, closeFn
}

func initDBUp() (*sqlx.DB, goqu.DialectWrapper, string, func()) {
	dbConn, dialect, dbName, closeFn := initDB()



	return dbConn, dialect, dbName, closeFn
}
