package test

import (
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db/migrations"
	"github.com/doug-martin/goqu/v9"
	// imported here to ensure it's available for tests
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"sync"
	"time"
)

// LoadTestTimeZone returns the configured
// timezone or uses UTC if none is set.
func LoadTestTimeZone() (*time.Location, error) {
	var v string
	var found bool
	if v, found = os.LookupEnv("AIRTRACK_TEST_TIMEZONE"); !found {
		v = "UTC"
	}
	l, err := time.LoadLocation(v)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// MustLoadTestTimeZone loads a timezone using LoadTestTimeZone
// but will panic if an error is returned
func MustLoadTestTimeZone() *time.Location {
	tz, err := LoadTestTimeZone()
	if err != nil {
		panic(err)
	}
	return tz
}

// DbConfig defines information about the test database
type DbConfig struct {
	Driver      string
	DatabaseFmt string
	Username    string
	Password    string
	Host        string
	Port        int
	NumDbs      int
}

// LoadTestDbConfig can be called to create a database configuration
// based on environment variables.
func LoadTestDbConfig() (*config.Database, error) {
	cfg := &config.Database{
		Driver:   "mysql",
		Username: "root",
		Password: "",
		Host:     "127.0.0.1",
		Port:     3306,
	}
	if v, found := os.LookupEnv("AIRTRACK_TEST_DB_DRIVER"); found {
		cfg.Driver = v
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
	return cfg, nil
}

// DB contains structures for a test database
type DB struct {
	conf  *DbConfig
	tz    *time.Location
	db    string
	inUse bool
}

var dbs []*DB

var claimNextDBLock = &sync.Mutex{}

func init() {
	tz := MustLoadTestTimeZone()

	dbs = make([]*DB, 4)
	for i := 0; i < 4; i++ {
		dbs[i] = &DB{
			db:    fmt.Sprintf("airtrack_test_%d", i),
			tz:    tz,
			inUse: false}
	}
}

// ClaimNextDB returns the first free database. You should
// limit concurrency to the number of DB's you want to use,
// otherwise this function will panic.
func ClaimNextDB() *DB {
	claimNextDBLock.Lock()
	defer claimNextDBLock.Unlock()

	for _, testDB := range dbs {
		if !testDB.inUse {
			testDB.inUse = true
			return testDB
		}
	}

	panic("Failed to claim a DB")
}

// DropTablesPostgres deletes all tables in the database db
// is connected to
func DropTablesPostgres(db *sqlx.DB) error {
	rows, err := db.Query(`SELECT table_name
FROM information_schema.tables
WHERE table_schema='public'
AND table_type='BASE TABLE';`)
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

	for _, table := range tables {
		_, err = db.Exec("DROP TABLE " + table)
		if err != nil {
			return errors.Wrapf(err, "drop table")
		}
	}
	return nil
}

// DropTablesMysql deletes all tables in the database db
// is connected to
func DropTablesMysql(db *sqlx.DB) error {
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

	for _, table := range tables {
		_, err = db.Exec("DROP TABLE " + table)
		if err != nil {
			return errors.Wrapf(err, "drop table")
		}
	}
	return nil
}

func sqliteTempFile(testDB *DB) string {
	return "/tmp/" + testDB.db + ".sqlite3"
}
func initDb(testDB *DB, migrate bool) (*sqlx.DB, *config.Database, goqu.DialectWrapper, string, func()) {
	_, ok := os.LookupEnv("AIRTRACK_TEST_DB_DRIVER")
	var dbConf *config.Database
	var err error
	if ok {
		dbConf, err = LoadTestDbConfig()
		if err != nil {
			panic(err)
		}
		dbConf.Database = testDB.db
	} else {
		dbConf = &config.Database{
			Driver:   "sqlite3",
			Database: sqliteTempFile(testDB),
		}
	}

	dbURL, err := dbConf.DataSource(testDB.tz)
	if err != nil {
		panic(err)
	}
	if dbConf.Driver == config.DatabaseDriverSqlite3 {
		if _, err := os.Stat(dbConf.Database); err == nil {
			err = os.Remove(dbConf.Database)
			if err != nil {
				panic(err)
			}
		}
	} else if dbConf.Driver == config.DatabaseDriverPostgresql {
		// only need this for circleci really
		dbURL = dbURL + " sslmode=disable"
	}

	sqlConn, err := sql.Open(dbConf.Driver, dbURL)
	if err != nil {
		panic(err)
	}
	dbConn := sqlx.NewDb(sqlConn, dbConf.Driver)

	err = dbConn.Ping()
	if err != nil {
		panic(err)
	}

	if migrate {
		switch dbConf.Driver {
		case config.DatabaseDriverMySQL:
			err = DropTablesMysql(dbConn)
			if err != nil {
				panic(err)
			}
		case config.DatabaseDriverPostgresql:
			err = DropTablesPostgres(dbConn)
			if err != nil {
				panic(err)
			}
		}

		m, err := migrations.InitMigrations(dbConf, testDB.tz)
		if err != nil {
			panic(err)
		}
		err = m.Up()
		if err != nil {
			panic(err)
		}
	}

	// create close function
	closeFn := func() {
		dbConn.Close()
		testDB.inUse = false
	}
	dialect := goqu.Dialect(dbConf.Driver)

	return dbConn, dbConf, dialect, testDB.db, closeFn
}

// InitDB claims a DB and creates a connection. The resulting database
// will NOT be initialized with migrations.
func InitDB() (*sqlx.DB, *config.Database, goqu.DialectWrapper, string, func()) {
	testDB := ClaimNextDB()
	dbConn, dbConf, dialect, dbName, closeFn := initDb(testDB, false)
	return dbConn, dbConf, dialect, dbName, closeFn
}

// InitDBUp claims a DB and creates a connection. The resulting database
// will be initialized with migrations.
func InitDBUp() (*sqlx.DB, goqu.DialectWrapper, string, func()) {
	testDB := ClaimNextDB()
	dbConn, _, dialect, dbName, closeFn := initDb(testDB, true)
	return dbConn, dialect, dbName, closeFn
}
