package tracker

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/test"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"strings"
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
	dbConf := test.MustLoadTestDbConfig()
	tz := test.MustLoadTestTimeZone()

	if !strings.Contains(dbConf.DatabaseFmt, "%d") {
		panic(errors.Errorf("database name (%s) must contain '%%d'", dbConf.DatabaseFmt))
	}

	dbs = make([]*TestDB, dbConf.NumDbs)
	for i := 0; i < dbConf.NumDbs; i++ {
		dbs[i] = &TestDB{conf: dbConf,
			db:    fmt.Sprintf(dbConf.DatabaseFmt, i),
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

func initDB() (*sqlx.DB, string, func()) {
	testDB := claimNextDB()
	dbConf := testDB.conf
	dbConn, err := db.NewMultiStmtConn(dbConf.Driver, dbConf.Username, dbConf.Password, dbConf.Host, dbConf.Port, testDB.db, testDB.tz)
	if err != nil {
		panic(fmt.Sprintf("error creating database connection %s", err.Error()))
	}

	err = dropTables(dbConn)
	if err != nil {
		panic(err)
	}

	// create close function
	closeFn := func() {
		dbConn.Close()
		testDB.inUse = false
	}

	return dbConn, testDB.db, closeFn
}

func initDBUp() (*sqlx.DB, string, func()) {
	dbConn, dbName, closeFn := initDB()

	migrate, err := test.InitMigration(dbName, dbConn.DB)
	if err != nil {
		panic(err)
	}
	err = migrate.Up()
	if err != nil {
		panic(err)
	}

	return dbConn, dbName, closeFn
}
