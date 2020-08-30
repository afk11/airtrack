package config

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"net/url"
	"os"
	"time"
)

const (
	// MailDriverSmtp - the only support SMTP driver
	MailDriverSmtp           = "smtp"
	DatabaseDriverMySQL      = "mysql"
	DatabaseDriverPostgresql = "postgresql"
	DatabaseDriverSqlite3    = "sqlite3"
)

type (
	// Airports contains configuration for airport geolocation
	Airports struct {
		// OpenAIPDirectories defined here will be scanned for .aip files
		OpenAIPDirectories []string `yaml:"openaip"`
		// CupDirecories defined here will be scanned for .cup files
		CupDirectories []string `yaml:"cup"`
	}

	// SmtpSettings - contains connection information for SMTP server.
	SmtpSettings struct {
		// Username - the SMTP username
		Username string `yaml:"username"`
		// Password - the SMTP password
		Password string `yaml:"password"`
		// Sender - the originating email address
		Sender string `yaml:"sender"`
		// Host - the SMTP server's hostname/ip
		Host string `yaml:"host"`
		// Port - the SMTP server port
		Port int `yaml:"port"`
		// TLS - whether to connect using TLS (port 587)
		TLS bool `yaml:"tls"`
		// MandatoryStartTLS - if set to true, connections to servers
		// which do not advertise STARTTLS support will cause an error.
		MandatoryStartTLS bool `yaml:"mandatory_starttls"`
		// NoStartTLS set to true disables opportunistic STARTTLS behaviour,
		// where the connection will be completely plaintext
		NoStartTLS bool `yaml:"nostarttls"`
	}

	// MapSettings contains configuration for providing
	// aircraft maps
	MapSettings struct {
		// Toggles whether map is enabled (default FALSE)
		Disabled bool `yaml:"disabled"`
		// HistoryInterval - number of seconds between new history files
		// (default: 30 seconds)
		HistoryInterval int64 `yaml:"history_interval"`
		// HistoryCount worth of history files will be kept. (default: 60)
		HistoryCount int `yaml:"history_count"`
		// Map interfaces to expose (default: dump1090 + tar1090)
		Services []string `yaml:"services"`
		// Address webserver should listen on.
		Address string `yaml:"address"`
		// Port webserver should listen on (default: 8080)
		Port uint16 `yaml:"port"`
	}

	// EmailSettings is where email support is configured
	EmailSettings struct {
		// Driver - currently only 'smtp' is supported
		Driver string `yaml:"driver"`
		// Smtp points to a SmtpSettings struct for use with
		// the 'smtp' driver
		Smtp *SmtpSettings `yaml:"smtp"`
	}

	// Notifications - contains configuration of events to
	// send to user
	Notifications struct {
		Email   string   `yaml:"email"`
		Enabled []string `yaml:"events"`
	}

	// ProjectMapSettings contains project level configuration
	// for the HTTP map UI
	ProjectMapSettings struct {
		// Disabled controls whether this project's map shall be
		// displayed on the HTTP map server. (default: false)
		Disabled bool `yaml:"disabled"`
	}

	// Project contains configuration for a single project
	Project struct {
		// Name - the name of the project (required)
		Name string `yaml:"name"`
		// Disabled controls whether the project should be running or not
		// this session. (default: false)
		Disabled bool `yaml:"disabled"`
		// Filter - an optional filter to apply to incoming messages
		Filter string
		// Map contains project level configuration for the map UI
		Map *ProjectMapSettings `yaml:"map"`
		// Notifications - per project configuration of event notifications
		Notifications *Notifications `yaml:"notifications"`
		// Features - per project extra features
		Features []string
		// ReopenSightings - whether to reopen a previously closed sighting
		// if a new sighting is within a certain timeframe
		ReopenSightings bool `yaml:"reopen_sightings"`
		// ReopenSightingsInterval - How long after an aircraft goes out of range
		// before we no longer reopen a recently closed session. Default 5m.
		ReopenSightingsInterval int `yaml:"reopen_sightings_interval"`
		// OnGroundUpdateThreshold - how many on_ground messages before we propagate
		// the change in status
		OnGroundUpdateThreshold *int64 `yaml:"onground_update_threshold"`
	}

	// Database - connection information about the database
	Database struct {
		// Driver to use for connections: sqlite3, mysql, postgresql
		Driver string `yaml:"driver"`
		// Host applies to mysql/postgresql - the DB server to connect to
		Host string `yaml:"host"`
		// Port applies to mysql/postgresql - the DB server port to connect to
		Port int `yaml:"port"`
		// Username is the username used when connecting to mysql/postgresql
		Username string `yaml:"username"`
		// Password is the password used when connecting to mysql/postgresql
		Password string `yaml:"password"`
		// Database - filesystem path to sqlite3 file, or database name on mysql/postgresql
		Database string `yaml:"database"`
	}

	Metrics struct {
		Enabled bool `yaml:"enabled"`
		Port    int  `yaml:"port"`
	}

	AdsbxConfig struct {
		// Custom ADSB Exchange URL (not required, but useful if
		// you've a proxy setup)
		ApiUrl string `yaml:"url"`
		// ADSB Exchange API key
		ApiKey string `yaml:"apikey"`
	}

	Config struct {
		TimeZone      *string        `yaml:"timezone"`
		AdsbxConfig   AdsbxConfig    `yaml:"adsbx"`
		Airports      Airports       `yaml:"airports"`
		EmailSettings *EmailSettings `yaml:"email"`
		Database      Database       `yaml:"database"`
		Metrics       *Metrics       `yaml:"metrics"`
		MapSettings   *MapSettings   `yaml:"map"`
		Sighting      struct {
			Timeout *int64 `yaml:"timeout"`
		} `yaml:"sighting"`
		Projects []Project `yaml:"projects"`
	}

	ProjectsConfig struct {
		Projects []Project `yaml:"projects"`
	}
)

func (c *Config) GetTimeLocation() (*time.Location, error) {
	// Use provided timezone, or use system timezone
	if c.TimeZone == nil {
		return time.Now().Location(), nil
	}

	tz := *c.TimeZone
	var err error
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid timezone `%s`", tz)
	}
	return loc, nil
}
func (d *Database) DataSource(loc *time.Location) (string, error) {
	switch d.Driver {
	case "":
		return "", errors.New("no database driver configured")
	case DatabaseDriverMySQL, DatabaseDriverPostgresql:
		return d.NetworkDatabaseUrl(loc)
	case DatabaseDriverSqlite3:
		return d.Sqlite3Url(loc)
	default:
		return "", errors.Errorf("unsupported database driver `%s`", d.Driver)
	}
}
func (d *Database) NetworkDatabaseUrl(location *time.Location) (string, error) {
	if d.Database == "" {
		return "", errors.New("database cannot be empty")
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s",
		d.Username, d.Password, d.Host, d.Port, d.Database, url.PathEscape(location.String())), nil
}
func (d *Database) Sqlite3Url(location *time.Location) (string, error) {
	if d.Database == "" {
		return "", errors.New("database filesystem path cannot be empty")
	}
	return fmt.Sprintf("file:%s?parseTime=true&loc=%s",
		d.Database, url.PathEscape(location.String())), nil
}

// ReadConfigFromFile will read `filepath` and attempt to parse into
// a Config structure. An error will be returned if duplicated project
// names are encountered.
func ReadConfigFromFile(filepath string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadConfig(f)
}

// ReadConfig will decode the provided reader into a Config structure.
// An error will be returned if duplicated project names are encountered.
func ReadConfig(r io.Reader) (*Config, error) {
	cfg := Config{}
	decoder := yaml.NewDecoder(r)
	err := decoder.Decode(&cfg)
	if err != nil {
		return &cfg, err
	}
	projMap := make(map[string]struct{})
	for _, proj := range cfg.Projects {
		_, ok := projMap[proj.Name]
		if ok {
			return nil, errors.Errorf("duplicated project name: %s", proj.Name)
		}
	}
	return &cfg, nil
}

// ReadConfigFromFile will read `filepath` and attempt to parse into
// a Config structure. An error will be returned if duplicated project
// names are encountered.
func ReadProjectsConfigFromFile(filepath string) (*ProjectsConfig, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadProjectsConfig(f)
}

// ReadConfig will decode the provided reader into a ProjectsConfig structure.
// An error will be returned if duplicated project names are encountered.
func ReadProjectsConfig(r io.Reader) (*ProjectsConfig, error) {
	cfg := ProjectsConfig{}
	decoder := yaml.NewDecoder(r)
	err := decoder.Decode(&cfg)
	if err != nil {
		return &cfg, err
	}
	projMap := make(map[string]struct{})
	for _, proj := range cfg.Projects {
		_, ok := projMap[proj.Name]
		if ok {
			return nil, errors.Errorf("duplicated project name: %s", proj.Name)
		}
	}
	return &cfg, nil
}

// ReadConfigs will decode the provided 'main' configFile, along with
// any extra project only files, and return the initialized configuration.
// An error will be returned if duplicated project names are encountered.
func ReadConfigs(configFile string, projectsFiles []string) (*Config, error) {
	if configFile == "" {
		return nil, errors.New("configuration file empty")
	}
	cfg, err := ReadConfigFromFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading main config file")
	}
	// build up current projects. ReadConfigFromFile guarantees no duplicates.
	projects := make(map[string]struct{})
	for _, proj := range cfg.Projects {
		projects[proj.Name] = struct{}{}
	}
	for _, projectsFile := range projectsFiles {
		projConfig, err := ReadProjectsConfigFromFile(projectsFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read projects config file %s", projectsFile)
		}
		for _, proj := range projConfig.Projects {
			// ensure no new duplicates
			if _, ok := projects[proj.Name]; ok {
				return nil, errors.Wrapf(err, "duplicated project name: %s", proj.Name)
			}
			cfg.Projects = append(cfg.Projects, proj)
		}
	}
	return cfg, nil
}
