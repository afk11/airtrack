package test

import (
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/db/migrations"
	"github.com/afk11/airtrack/pkg/db/migrations_sqlite3"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"
	"github.com/golang-migrate/migrate/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"os"
	"strconv"
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
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
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
		return nil, err
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}
	s := bindata.Resource(migrations_sqlite3.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations_sqlite3.Asset(name)
		})
	src, err := bindata.WithInstance(s)
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithInstance(
		"go-bindata",
		src,
		database,
		driver,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}
