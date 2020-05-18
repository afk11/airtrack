package email

import (
	"bytes"
	"fmt"
	asset "github.com/afk11/airtrack/pkg/assets"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/pkg/errors"
	"text/template"
	"time"
)

type Email string

func (e Email) String() string {
	return string(e)
}

const (
	MapProducedEmail Email = "assets/email/map_produced.tpl"
	SpottedInFlight  Email = "assets/email/spotted_in_flight.tpl"
)

var (
	EmailNotFoundErr = errors.Errorf("email template not found")
)

func GetTemplates() []Email {
	return []Email{
		MapProducedEmail,
		SpottedInFlight,
	}
}

type Location struct {
	Latitude  float64
	Longitude float64
	Altitude  int64
}
type SpottedInFlightParameters struct {
	Project       string
	Icao          string
	CallSign      string
	StartTime     time.Time
	StartTimeFmt  string
	StartLocation Location
}
type MapProducedParameters struct {
	Project       string
	Icao          string
	CallSign      string
	StartTime     time.Time
	EndTime       time.Time
	DurationFmt   string
	StartTimeFmt  string
	StartLocation Location
	EndTimeFmt    string
	EndLocation   Location
	MapUpdated    bool
}
type MailTemplates struct {
	m map[Email]*template.Template
}

func (t *MailTemplates) Get(email Email) (*template.Template, error) {
	tpl, ok := t.m[email]
	if !ok {
		return nil, EmailNotFoundErr
	}
	return tpl, nil
}
func LoadMailTemplates(templates ...Email) (*MailTemplates, error) {
	m := MailTemplates{
		m: make(map[Email]*template.Template),
	}
	for _, email := range templates {
		data, err := asset.Asset(email.String())
		if err != nil {
			return nil, err
		}
		tpl := template.New(email.String())
		tpl, err = tpl.Parse(string(data))
		if err != nil {
			return nil, err
		}
		m.m[email] = tpl
	}
	return &m, nil
}
func PrepareSpottedInFlightEmail(templates *MailTemplates, to string, params SpottedInFlightParameters) (*db.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: spotted in flight", params.Project, params.Icao, callsign)
	tpl, err := templates.Get(SpottedInFlight)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, params)
	if err != nil {
		return nil, err
	}
	job := &db.EmailJob{
		To:          to,
		Subject:     subject,
		Body:        buf.String(),
		Attachments: nil,
	}

	return job, nil
}
func PrepareMapProducedEmail(templates *MailTemplates, to string, kmlFile string, params MapProducedParameters) (*db.EmailJob, error) {
	action := "created"
	var callsign string
	if params.MapUpdated {
		action = "updated"
	}
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: flight map %s", params.Project, params.Icao, callsign, action)

	tpl, err := templates.Get(MapProducedEmail)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, params)
	if err != nil {
		return nil, err
	}
	job := &db.EmailJob{
		To:      to,
		Subject: subject,
		Body:    buf.String(),
		Attachments: []db.EmailAttachment{
			{
				Contents: kmlFile,
				FileName: fmt.Sprintf("%s-%s.kml",
					params.Icao, params.EndTimeFmt),
				ContentType: "application/vnd.google-earth.kml+xml",
			},
		},
	}
	return job, nil
}
