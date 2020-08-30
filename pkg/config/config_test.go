package config

import (
	"bytes"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestReadProjectsConfig(t *testing.T) {
	t.Run("projects", func(t *testing.T) {
		buf := bytes.NewBufferString(`
projects:
  - name: German aircraft
    filter: state.CountryCode == "DE"
    disabled: true
  - name: UK aircraft
    filter: state.CountryCode == "GB"
    reopen_sightings: true
    reopen_sightings_interval: 10
    notifications:
      email: email@domain.local
      events:
        - map_produced
        - spotted_in_flight
    features:
      - track_callsigns
      - track_squawks
      - track_takeoff
`)
		cfg, err := ReadProjectsConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, 2, len(cfg.Projects))

		assert.Equal(t, "German aircraft", cfg.Projects[0].Name)
		assert.Equal(t, `state.CountryCode == "DE"`, cfg.Projects[0].Filter)
		assert.False(t, cfg.Projects[0].ReopenSightings)
		assert.True(t, cfg.Projects[0].Disabled)
		assert.Nil(t, cfg.Projects[0].Notifications)
		assert.Nil(t, cfg.Projects[0].Features)

		assert.Equal(t, "UK aircraft", cfg.Projects[1].Name)
		assert.Equal(t, `state.CountryCode == "GB"`, cfg.Projects[1].Filter)
		assert.True(t, cfg.Projects[1].ReopenSightings)
		assert.Equal(t, 10, cfg.Projects[1].ReopenSightingsInterval)
		assert.False(t, cfg.Projects[1].Disabled)
		assert.Equal(t, "email@domain.local", cfg.Projects[1].Notifications.Email)
		assert.Equal(t, 2, len(cfg.Projects[1].Notifications.Enabled))
		assert.Equal(t, "map_produced", cfg.Projects[1].Notifications.Enabled[0])
		assert.Equal(t, "spotted_in_flight", cfg.Projects[1].Notifications.Enabled[1])
		assert.Equal(t, 3, len(cfg.Projects[1].Features))
		assert.Equal(t, "track_callsigns", cfg.Projects[1].Features[0])
		assert.Equal(t, "track_squawks", cfg.Projects[1].Features[1])
		assert.Equal(t, "track_takeoff", cfg.Projects[1].Features[2])
	})
}

func TestReadConfig(t *testing.T) {
	t.Run("airports", func(t *testing.T) {
		buf := bytes.NewBufferString(`
timezone: UTC
encryption:
  key: G7ZgLnbGr9YVI+w+rHEhs2MDtVxLI68AqMWv+9dl0zk=
airports:
  openaip:
    - ./dir1/
    - ./dir2/
`)
		cfg, err := ReadConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "UTC", *cfg.TimeZone, "parsed timezone should match")
		assert.Equal(t, 2, len(cfg.Airports.OpenAIPDirectories))
		assert.Equal(t, "./dir1/", cfg.Airports.OpenAIPDirectories[0])
		assert.Equal(t, "./dir2/", cfg.Airports.OpenAIPDirectories[1])
		assert.Nil(t, cfg.EmailSettings)
	})

	t.Run("metrics", func(t *testing.T) {
		buf := bytes.NewBufferString(`
timezone: UTC
encryption:
  key: G7ZgLnbGr9YVI+w+rHEhs2MDtVxLI68AqMWv+9dl0zk=
metrics:
  enabled: true
  port: 9999
`)
		cfg, err := ReadConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "UTC", *cfg.TimeZone, "parsed timezone should match")
		assert.NotNil(t, cfg.Metrics)
		assert.Equal(t, true, cfg.Metrics.Enabled)
		assert.Equal(t, 9999, cfg.Metrics.Port)
	})

	t.Run("sighting", func(t *testing.T) {
		buf := bytes.NewBufferString(`
timezone: UTC
encryption:
  key: G7ZgLnbGr9YVI+w+rHEhs2MDtVxLI68AqMWv+9dl0zk=
sighting:
  timeout: 60
`)
		cfg, err := ReadConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "UTC", *cfg.TimeZone, "parsed timezone should match")
		assert.NotNil(t, cfg.Sighting)
		assert.Equal(t, int64(60), *cfg.Sighting.Timeout)
	})

	t.Run("database", func(t *testing.T) {
		buf := bytes.NewBufferString(`
timezone: UTC
encryption:
  key: G7ZgLnbGr9YVI+w+rHEhs2MDtVxLI68AqMWv+9dl0zk=
database:
  driver: mysql
  host: server.local
  port: 3306
  username: airtrackuser
  password: password
  database: airtrackdb
`)
		cfg, err := ReadConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "UTC", *cfg.TimeZone, "parsed timezone should match")
		assert.NotNil(t, cfg.Database)
		assert.Equal(t, "mysql", cfg.Database.Driver)
		assert.Equal(t, "server.local", cfg.Database.Host)
		assert.Equal(t, 3306, cfg.Database.Port)
		assert.Equal(t, "airtrackuser", cfg.Database.Username)
		assert.Equal(t, "password", cfg.Database.Password)
		assert.Equal(t, "airtrackdb", cfg.Database.Database)
	})

	t.Run("smtp", func(t *testing.T) {
		buf := bytes.NewBufferString(`
timezone: UTC
encryption:
  key: G7ZgLnbGr9YVI+w+rHEhs2MDtVxLI68AqMWv+9dl0zk=
email:
  driver: smtp
  smtp:
    username: mylogin
    password: mypassword
    sender: email@website.local
    host: website.local
    port: 587
    mandatory_starttls: true
`)
		cfg, err := ReadConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "UTC", *cfg.TimeZone, "parsed timezone should match")
		assert.NotNil(t, cfg.EmailSettings)
		assert.Equal(t, "smtp", cfg.EmailSettings.Driver)
		assert.Equal(t, "mylogin", cfg.EmailSettings.Smtp.Username)
		assert.Equal(t, "mypassword", cfg.EmailSettings.Smtp.Password)
		assert.Equal(t, "email@website.local", cfg.EmailSettings.Smtp.Sender)
		assert.Equal(t, "website.local", cfg.EmailSettings.Smtp.Host)
		assert.Equal(t, 587, cfg.EmailSettings.Smtp.Port)
		assert.Equal(t, true, cfg.EmailSettings.Smtp.MandatoryStartTLS)
	})

	t.Run("projects", func(t *testing.T) {
		buf := bytes.NewBufferString(`
timezone: UTC
encryption:
  key: G7ZgLnbGr9YVI+w+rHEhs2MDtVxLI68AqMWv+9dl0zk=
projects:
  - name: German aircraft
    filter: state.CountryCode == "DE"
    disabled: true
  - name: UK aircraft
    filter: state.CountryCode == "GB"
    reopen_sightings: true
    reopen_sightings_interval: 10
    onground_update_threshold: 7
    notifications:
      email: email@domain.local
      events:
        - map_produced
        - spotted_in_flight
    features:
      - track_callsigns
      - track_squawks
      - track_takeoff
`)
		cfg, err := ReadConfig(buf)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "UTC", *cfg.TimeZone, "parsed timezone should match")
		assert.Equal(t, 2, len(cfg.Projects))

		assert.Equal(t, "German aircraft", cfg.Projects[0].Name)
		assert.Equal(t, `state.CountryCode == "DE"`, cfg.Projects[0].Filter)
		assert.False(t, cfg.Projects[0].ReopenSightings)
		assert.True(t, cfg.Projects[0].Disabled)
		assert.Nil(t, cfg.Projects[0].Notifications)
		assert.Nil(t, cfg.Projects[0].Features)

		assert.Equal(t, "UK aircraft", cfg.Projects[1].Name)
		assert.Equal(t, `state.CountryCode == "GB"`, cfg.Projects[1].Filter)
		assert.True(t, cfg.Projects[1].ReopenSightings)
		assert.Equal(t, 10, cfg.Projects[1].ReopenSightingsInterval)
		assert.Equal(t, int64(7), *cfg.Projects[1].OnGroundUpdateThreshold)
		assert.False(t, cfg.Projects[1].Disabled)
		assert.Equal(t, "email@domain.local", cfg.Projects[1].Notifications.Email)
		assert.Equal(t, 2, len(cfg.Projects[1].Notifications.Enabled))
		assert.Equal(t, "map_produced", cfg.Projects[1].Notifications.Enabled[0])
		assert.Equal(t, "spotted_in_flight", cfg.Projects[1].Notifications.Enabled[1])
		assert.Equal(t, 3, len(cfg.Projects[1].Features))
		assert.Equal(t, "track_callsigns", cfg.Projects[1].Features[0])
		assert.Equal(t, "track_squawks", cfg.Projects[1].Features[1])
		assert.Equal(t, "track_takeoff", cfg.Projects[1].Features[2])
	})
}

func TestReadConfigs(t *testing.T) {
	t.Run("empty config file", func(t *testing.T) {
		_, err := ReadConfigs("", nil)
		assert.EqualError(t, err, "configuration file empty")
	})
}
