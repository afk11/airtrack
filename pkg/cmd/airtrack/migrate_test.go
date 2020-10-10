package airtrack

import (
	"database/sql"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"testing"
)

func writeConfigToFile(cfg *config.Config) (string, error) {
	fh, err := ioutil.TempFile("/tmp", "airtrack-config")
	if err != nil {
		return "", err
	}
	encoder := yaml.NewEncoder(fh)
	err = encoder.Encode(cfg)
	if err != nil {
		os.Remove(fh.Name())
		return "", err
	}
	return fh.Name(), nil
}
func TestMigrateUpAndDown(t *testing.T) {
	dbConn, dbConf, dialect, _, closer := test.InitDB()
	defer closer()

	cfg := &config.Config{}
	cfg.Database = *dbConf

	configPath, err := writeConfigToFile(cfg)
	assert.NoError(t, err)

	defer os.Remove(configPath)

	d := db.NewDatabase(dbConn, dialect)
	sm, err := d.GetSchemaMigration()
	assert.Error(t, err)

	up := MigrateUpCmd{
		Config: configPath,
		Force:  true,
	}
	err = up.Run()
	assert.NoError(t, err)

	sm, err = d.GetSchemaMigration()
	assert.NoError(t, err)
	assert.NotNil(t, sm)
	assert.Greater(t, sm.Version, uint64(8))

	down := MigrateDownCmd{
		Config: configPath,
		Force:  true,
	}
	err = down.Run()
	assert.NoError(t, err)

	sm, err = d.GetSchemaMigration()
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}
