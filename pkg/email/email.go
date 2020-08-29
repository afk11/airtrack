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

var (
	EmailNotFoundErr = errors.Errorf("email template not found")
)

type (
	Email string

	Location struct {
		Latitude  float64
		Longitude float64
		Altitude  int64
	}
	SpottedInFlightParameters struct {
		Project       string
		Icao          string
		CallSign      string
		StartTime     time.Time
		StartTimeFmt  string
		StartLocation Location
	}
	MapProducedParameters struct {
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
	TakeoffParams struct {
		Project       string
		Icao          string
		CallSign      string
		AirportName   string
		StartTimeFmt  string
		StartLocation Location
	}
	TakeoffCompleteParams struct {
		Project       string
		Icao          string
		CallSign      string
		AirportName   string
		StartTimeFmt  string
		StartLocation Location
	}
	TakeoffUnknownAirportParams struct {
		Project       string
		Icao          string
		CallSign      string
		StartTimeFmt  string
		StartLocation Location
	}
	MailTemplates struct {
		m map[Email]*template.Template
	}
)

const (
	MapProducedEmail      Email = "assets/email/map_produced.tpl"
	SpottedInFlight       Email = "assets/email/spotted_in_flight.tpl"
	TakeoffUnknownAirport Email = "assets/email/takeoff_unknown_airport.tpl"
	TakeoffFromAirport    Email = "assets/email/takeoff_from_airport.tpl"
	TakeoffComplete       Email = "assets/email/takeoff_complete.tpl"
)

func GetTemplates() []Email {
	return []Email{
		MapProducedEmail,
		SpottedInFlight,
		TakeoffUnknownAirport,
		TakeoffFromAirport,
		TakeoffComplete,
	}
}

func (e Email) String() string {
	return string(e)
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

func buildEmail(templates *MailTemplates, email Email, to string, subject string, params interface{}) (*db.EmailJob, error) {
	tpl, err := templates.Get(email)
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
	}
	return job, nil
}

func buildEmailWithAttachment(templates *MailTemplates, email Email, to string, subject string, params interface{}, attachments []db.EmailAttachment) (*db.EmailJob, error) {
	tpl, err := templates.Get(email)
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
		Attachments: attachments,
	}
	return job, nil
}

func PrepareSpottedInFlightEmail(templates *MailTemplates, to string, params SpottedInFlightParameters) (*db.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: spotted in flight", params.Project, params.Icao, callsign)
	return buildEmail(templates, SpottedInFlight, to, subject, params)
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
	return buildEmailWithAttachment(templates, MapProducedEmail, to, subject, params, []db.EmailAttachment{
		{
			Contents: kmlFile,
			FileName: fmt.Sprintf("%s-%s.kml",
				params.Icao, params.EndTimeFmt),
			ContentType: "application/vnd.google-earth.kml+xml",
		},
	})
}

func PrepareTakeoffFromAirport(templates *MailTemplates, to string, params TakeoffParams) (*db.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: takeoff from %s", params.Project, params.Icao, callsign, params.AirportName)

	return buildEmail(templates, TakeoffFromAirport, to, subject, params)
}

func PrepareTakeoffUnknownAirport(templates *MailTemplates, to string, params TakeoffUnknownAirportParams) (*db.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: takeoff from unknown airport", params.Project, params.Icao, callsign)

	return buildEmail(templates, TakeoffUnknownAirport, to, subject, params)
}

func PrepareTakeoffComplete(templates *MailTemplates, to string, params TakeoffCompleteParams) (*db.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: takeoff complete", params.Project, params.Icao, callsign)

	return buildEmail(templates, TakeoffComplete, to, subject, params)
}
