package test

import (
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/db/migrations_mysql"
	"github.com/afk11/airtrack/pkg/db/migrations_sqlite3"
	"github.com/doug-martin/goqu/v9"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"
	"github.com/golang-migrate/migrate/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"sync"
	"time"
)

type TestTimeZone struct {
	Tz *time.Location
}

func LoadTestTimeZone() (*TestTimeZone, error) {
	var v string
	var found bool
	if v, found = os.LookupEnv("AIRTRACK_TEST_TIMEZONE"); !found {
		v = "UTC"
	}
	l, err := time.LoadLocation(v)
	if err != nil {
		return nil, err
	}
	return &TestTimeZone{
		Tz: l,
	}, nil
}
func MustLoadTestTimeZone() *TestTimeZone {
	tz, err := LoadTestTimeZone()
	if err != nil {
		panic(err)
	}
	return tz
}

type TestDbConfig struct {
	Driver      string
	DatabaseFmt string
	Username    string
	Password    string
	Host        string
	Port        int
	NumDbs      int
}

func MustLoadTestDbConfig() *TestDbConfig {
	c, err := LoadTestDbConfig()
	if err != nil {
		panic(err)
	}
	return c
}
func LoadTestDbConfig() (*TestDbConfig, error) {
	cfg := &TestDbConfig{
		Driver:      "mysql",
		DatabaseFmt: "airtrack_test_%d",
		Username:    "root",
		Password:    "",
		Host:        "127.0.0.1",
		Port:        3306,
		NumDbs:      1,
	}
	if v, found := os.LookupEnv("AIRTRACK_TEST_DB_USER"); found {
		cfg.Username = v
	}
	if v, found := os.LookupEnv("AIRTRACK_TEST_DB_PASS"); found {
		cfg.Password = v
	}
	if v, found := os.LookupEnv("AIRTRACK_TEST_DB_HOST"); found {
		cfg.Host = v
	}
	if v, found := os.LookupEnv("AIRTRACK_TEST_DB_PORT"); found {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		cfg.Port = p
	}
	if v, found := os.LookupEnv("AIRTRACK_TEST_DB_NUM_DBS"); found {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		cfg.NumDbs = n
	}
	return cfg, nil
}

func InitMysqlMigration(database string, db *sql.DB) (*migrate.Migrate, error) {
	s := bindata.Resource(migrations_mysql.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations_mysql.Asset(name)
		})
	d, err := bindata.WithInstance(s)
	if err != nil {
		return nil, err
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithInstance(
		"go-bindata",
		d,
		database,
		driver,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}
func InitSqliteMigration(database string, sqliteFile string, db *sql.DB) (*migrate.Migrate, error) {
	dbUrl := fmt.Sprintf("file:" + sqliteFile)
	db, err := sql.Open("sqlite3", dbUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "opening sqlite file")
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, errors.Wrapf(err, "setup migrate sqlite3 driver")
	}
	s := bindata.Resource(migrations_sqlite3.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations_sqlite3.Asset(name)
		})
	src, err := bindata.WithInstance(s)
	if err != nil {
		return nil, errors.Wrapf(err, "bindata source with instance")
	}
	m, err := migrate.NewWithInstance(
		"go-bindata",
		src,
		database,
		driver,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "init migrate instance")
	}
	return m, nil
}

type TestDB struct {
	conf  *TestDbConfig
	tz    *time.Location
	db    string
	inUse bool
}

var dbs []*TestDB

var claimNextDBLock = &sync.Mutex{}

func init() {
	tz := MustLoadTestTimeZone()

	dbs = make([]*TestDB, 4)
	for i := 0; i < 4; i++ {
		dbs[i] = &TestDB{
			db:    fmt.Sprintf("airtrack_%d", i),
			tz:    tz.Tz,
			inUse: false}
	}
}

func ClaimNextDB() *TestDB {
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

func DropTables(db *sqlx.DB) error {
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

func InitDB() (*sqlx.DB, goqu.DialectWrapper, string, func()) {
	testDB := ClaimNextDB()

	tmpFile := "/tmp/" + testDB.db + ".sqlite3"
	if _, err := os.Stat(tmpFile); err == nil {
		err = os.Remove(tmpFile)
		if err != nil {
			panic(err)
		}
	}

	dbUrl := fmt.Sprintf("file:" + tmpFile)
	sqlConn, err := sql.Open("sqlite3", dbUrl)
	dbConn := sqlx.NewDb(sqlConn, "sqlite3")
	if err != nil {
		panic(fmt.Sprintf("error creating database connection %s", err.Error()))
	}

	err = dbConn.Ping()
	if err != nil {
		panic(err)
	}

	m, err := InitSqliteMigration(testDB.db, tmpFile, dbConn.DB)
	if err != nil {
		panic(err)
	}
	err = m.Up()
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

func InitDBUp() (*sqlx.DB, goqu.DialectWrapper, string, func()) {
	dbConn, dialect, dbName, closeFn := InitDB()

	return dbConn, dialect, dbName, closeFn
}
