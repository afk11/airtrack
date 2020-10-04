package email

import (
	"bytes"
	"fmt"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/pkg/errors"
	"text/template"
	"time"
)

var (
	// TemplateNotFoundErr is returned by MailTemplates if the
	// requested template cannot be found
	TemplateNotFoundErr = errors.Errorf("email template not found")
)

type (
	// Email - data type used to identify an email template
	Email string

	// Location - a structure containing an aircraft's location
	Location struct {
		// Latitude - decimal latitude
		Latitude float64
		// Longitude - decimal longitude
		Longitude float64
		// Altitude - height in ft
		// todo: Units
		Altitude int64
	}

	// SpottedInFlightParameters contains parameters for the
	// SpottedInFlight template.
	SpottedInFlightParameters struct {
		Project       string
		Icao          string
		CallSign      string
		StartTime     time.Time
		StartTimeFmt  string
		StartLocation Location
	}

	// MapProducedParameters contains parameters for the
	// MapProducedParameters template.
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

	// TakeoffParams contains parameters for the
	// TakeoffParams template.
	TakeoffParams struct {
		Project       string
		Icao          string
		CallSign      string
		AirportName   string
		StartTimeFmt  string
		StartLocation Location
	}

	// TakeoffCompleteParams contains parameters for the
	// TakeoffCompleteParams template.
	TakeoffCompleteParams struct {
		Project       string
		Icao          string
		CallSign      string
		AirportName   string
		StartTimeFmt  string
		StartLocation Location
	}

	// TakeoffUnknownAirportParams contains parameters for the
	// TakeoffUnknownAirportParams template.
	TakeoffUnknownAirportParams struct {
		Project       string
		Icao          string
		CallSign      string
		StartTimeFmt  string
		StartLocation Location
	}

	// MailTemplates - map of Emails to parsed template
	MailTemplates struct {
		m map[Email]*template.Template
	}
)

const (
	// MapProducedEmail - the template's name
	MapProducedEmail Email = "map_produced.tpl"
	// SpottedInFlight - the template's name
	SpottedInFlight Email = "spotted_in_flight.tpl"
	// TakeoffUnknownAirport - the template's name
	TakeoffUnknownAirport Email = "takeoff_unknown_airport.tpl"
	// TakeoffFromAirport - the template's name
	TakeoffFromAirport Email = "takeoff_from_airport.tpl"
	// TakeoffComplete - the template's name
	TakeoffComplete Email = "takeoff_complete.tpl"
)

// GetTemplates returns a list of all known templates
func GetTemplates() []Email {
	return []Email{
		MapProducedEmail,
		SpottedInFlight,
		TakeoffUnknownAirport,
		TakeoffFromAirport,
		TakeoffComplete,
	}
}

// String - implements Stringer. Returns the template name.
func (e Email) String() string {
	return string(e)
}

// Get returns the template for Email if known, or an TemplateNotFoundErr
// if the Email is not known.
func (t *MailTemplates) Get(email Email) (*template.Template, error) {
	tpl, ok := t.m[email]
	if !ok {
		return nil, TemplateNotFoundErr
	}
	return tpl, nil
}

// LoadMailTemplates takes a list of Emails, loads and parses the template,
// initializing MapTemplates, or an error if one occurred.
func LoadMailTemplates(templates ...Email) (*MailTemplates, error) {
	m := MailTemplates{
		m: make(map[Email]*template.Template),
	}
	for _, email := range templates {
		data, err := Asset(email.String())
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

// buildEmail loads and builds the template specified by `email`,
// and returns a mailer.EmailJob payload
func buildEmail(templates *MailTemplates, email Email, to string, subject string, params interface{}) (*mailer.EmailJob, error) {
	tpl, err := templates.Get(email)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, params)
	if err != nil {
		return nil, err
	}
	job := &mailer.EmailJob{
		To:      to,
		Subject: subject,
		Body:    buf.String(),
	}
	return job, nil
}

// buildEmailWithAttachment loads and builds the template specified
// by `email`, and returns a mailer.EmailJob payload including attachments.
func buildEmailWithAttachment(templates *MailTemplates, email Email, to string, subject string, params interface{}, attachments []mailer.EmailAttachment) (*mailer.EmailJob, error) {
	tpl, err := templates.Get(email)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, params)
	if err != nil {
		return nil, err
	}
	job := &mailer.EmailJob{
		To:          to,
		Subject:     subject,
		Body:        buf.String(),
		Attachments: attachments,
	}
	return job, nil
}

// PrepareSpottedInFlightEmail creates an SpottedInFlight and returns a mailer.EmailJob
// for the email
func PrepareSpottedInFlightEmail(templates *MailTemplates, to string, params SpottedInFlightParameters) (*mailer.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: spotted in flight", params.Project, params.Icao, callsign)
	return buildEmail(templates, SpottedInFlight, to, subject, params)
}

// PrepareMapProducedEmail creates an MapProducedEmail and returns a mailer.EmailJob
// for the email with the KML attachment.
func PrepareMapProducedEmail(templates *MailTemplates, to string, kmlFile []byte, params MapProducedParameters) (*mailer.EmailJob, error) {
	action := "created"
	var callsign string
	if params.MapUpdated {
		action = "updated"
	}
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: flight map %s", params.Project, params.Icao, callsign, action)
	return buildEmailWithAttachment(templates, MapProducedEmail, to, subject, params, []mailer.EmailAttachment{
		{
			Contents: kmlFile,
			FileName: fmt.Sprintf("%s-%s.kml",
				params.Icao, params.EndTimeFmt),
			ContentType: "application/vnd.google-earth.kml+xml",
		},
	})
}

// PrepareTakeoffFromAirport creates an TakeoffFromAirport and returns a mailer.EmailJob
// for the email
func PrepareTakeoffFromAirport(templates *MailTemplates, to string, params TakeoffParams) (*mailer.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: takeoff from %s", params.Project, params.Icao, callsign, params.AirportName)

	return buildEmail(templates, TakeoffFromAirport, to, subject, params)
}

// PrepareTakeoffUnknownAirport creates an TakeoffUnknownAirport and returns a mailer.EmailJob
// for the email
func PrepareTakeoffUnknownAirport(templates *MailTemplates, to string, params TakeoffUnknownAirportParams) (*mailer.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: takeoff from unknown airport", params.Project, params.Icao, callsign)

	return buildEmail(templates, TakeoffUnknownAirport, to, subject, params)
}

// PrepareTakeoffComplete creates an TakeoffComplete and returns a mailer.EmailJob
// for the email
func PrepareTakeoffComplete(templates *MailTemplates, to string, params TakeoffCompleteParams) (*mailer.EmailJob, error) {
	var callsign string
	if params.CallSign != "" {
		callsign = " (" + params.CallSign + ")"
	}

	subject := fmt.Sprintf("[%s] %s%s: takeoff complete", params.Project, params.Icao, callsign)

	return buildEmail(templates, TakeoffComplete, to, subject, params)
}
