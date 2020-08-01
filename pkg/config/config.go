package config

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"os"
)

const (
	MailDriverSmtp = "smtp"
)

type (
	Airports struct {
		OpenAIPDirectories []string `yaml:"openaip"`
		CupDirectories     []string `yaml:"cup"`
	}

	SmtpSettings struct {
		Username          string `yaml:"username"`
		Password          string `yaml:"password"`
		Sender            string `yaml:"sender"`
		Host              string `yaml:"host"`
		Port              int    `yaml:"port"`
		MandatoryStartTLS bool   `yaml:"mandatory_starttls"`
	}

	MapSettings struct {
		// Toggles whether map is enabled (default FALSE)
		Enabled         bool  `yaml:"enabled"`
		HistoryInterval int64 `yaml:"history_interval"`
		// HistoryCount worth of history files will be kept. (default: 60)
		HistoryCount int `yaml:"history_count"`
		// Map interfaces to expose (default: dump1090 if none set)
		Services []string `yaml:"services"`
		// Address webserver should listen on.
		Address string `yaml:"address"`
		// Port webserver should listen on (default: 8080)
		Port uint16 `yaml:"port"`
	}

	EmailSettings struct {
		Driver string        `yaml:"driver"`
		Smtp   *SmtpSettings `yaml:"smtp"`
	}

	Notifications struct {
		Email   string   `yaml:"email"`
		Enabled []string `yaml:"events"`
	}

	ProjectMap struct {
		Enabled bool `yaml:"enabled"`
	}

	Project struct {
		Name string
		//
		Disabled                bool `yaml:"disabled"`
		Filter                  string
		Map                     ProjectMap     `yaml:"map"`
		Notifications           *Notifications `yaml:"notifications"`
		Features                []string
		ReopenSightings         bool   `yaml:"reopen_sightings"`
		ReopenSightingsInterval int    `yaml:"reopen_sightings_interval"`
		OnGroundUpdateThreshold *int64 `yaml:"onground_update_threshold"`
	}

	Database struct {
		Driver   string `yaml:"driver"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	}

	StaticLocation struct {
		Address   string  `yaml:"address"`
		Latitude  float64 `yaml:"latitude"`
		Longitude float64 `yaml:"longitude"`
		Radius    int64   `yaml:"radius"`
	}

	Metrics struct {
		Enabled bool `yaml:"enabled"`
		Port    int  `yaml:"port"`
	}

	Encryption struct {
		Key string `yaml:"key"`
	}

	AdsbxConfig struct {
		ApiUrl string `yaml:"url"`
		ApiKey string `yaml:"apikey"`
	}

	Config struct {
		TimeZone    string      `yaml:"timezone"`
		Encryption  Encryption  `yaml:"encryption"`
		Airports    Airports    `yaml:"airports"`
		Metrics     *Metrics    `yaml:"metrics"`
		AdsbxConfig AdsbxConfig `yaml:"adsbx"`
		MapSettings MapSettings `yaml:"map"`
		Sighting    struct {
			Timeout *int64 `yaml:"timeout"`
		} `yaml:"sighting"`
		Database      Database       `yaml:"database"`
		EmailSettings *EmailSettings `yaml:"email"`
		Projects      []Project      `yaml:"projects"`
	}

	ProjectsConfig struct {
		Projects []Project `yaml:"projects"`
	}
)

// ReadConfigFromFile will read `filepath` and attempt to parse into
// a Config structure. This function guarantees that an error will be
// returned if duplicated project names are encountered.
func ReadConfigFromFile(filepath string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadConfig(f)
}

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
// a Config structure. This function guarantees that an error will be
// returned if duplicated project names are encountered.
func ReadProjectsConfigFromFile(filepath string) (*ProjectsConfig, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadProjectsConfig(f)
}

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
