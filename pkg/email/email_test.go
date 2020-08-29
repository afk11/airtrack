package email

import (
	assert "github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestGetTemplates(t *testing.T) {
	tpl := GetTemplates()
	assert.Equal(t, 5, len(tpl))
	assert.Equal(t, MapProducedEmail, tpl[0])
	assert.Equal(t, SpottedInFlight, tpl[1])
	assert.Equal(t, TakeoffUnknownAirport, tpl[2])
	assert.Equal(t, TakeoffFromAirport, tpl[3])
	assert.Equal(t, TakeoffComplete, tpl[4])
}

func TestLoadMailTemplates(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		tpls, err := LoadMailTemplates(GetTemplates()...)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(tpls.m))
		mapProduced, err := tpls.Get(MapProducedEmail)
		assert.NoError(t, err)
		assert.NotNil(t, mapProduced)
		assert.Equal(t, string(MapProducedEmail), mapProduced.Name())

		spottedInFlight, err := tpls.Get(SpottedInFlight)
		assert.NoError(t, err)
		assert.NotNil(t, spottedInFlight)
		assert.Equal(t, string(SpottedInFlight), spottedInFlight.Name())
	})

	t.Run("unknown", func(t *testing.T) {
		tpls, err := LoadMailTemplates(GetTemplates()...)
		assert.NoError(t, err)
		_, err = tpls.Get("unknown")
		assert.EqualError(t, err, EmailNotFoundErr.Error())
	})
}

func TestPrepareSpottedInFlightEmail(t *testing.T) {
	t.Run("with callsign", func(t *testing.T) {
		tpls, err := LoadMailTemplates(GetTemplates()...)
		assert.NoError(t, err)
		job, err := PrepareSpottedInFlightEmail(tpls, "dest@site.local", SpottedInFlightParameters{
			Project:  "MyCoolProject",
			Icao:     "010101",
			CallSign: "AF1",
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(job.Attachments))
		assert.Equal(t, "dest@site.local", job.To)
		assert.Equal(t, "[MyCoolProject] 010101 (AF1): spotted in flight", job.Subject)
		assert.True(t, strings.Contains(job.Body, "Project: MyCoolProject"))
		assert.True(t, strings.Contains(job.Body, "010101"))
		assert.True(t, strings.Contains(job.Body, "AF1"))
		assert.True(t, strings.Contains(job.Body, "Time"))
		assert.True(t, strings.Contains(job.Body, "Place"))
	})
	t.Run("without callsign", func(t *testing.T) {
		tpls, err := LoadMailTemplates(GetTemplates()...)
		assert.NoError(t, err)
		job, err := PrepareSpottedInFlightEmail(tpls, "dest@site.local", SpottedInFlightParameters{
			Project: "MyCoolProject",
			Icao:    "010101",
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(job.Attachments))
		assert.Equal(t, "dest@site.local", job.To)
		assert.True(t, strings.Contains(job.Body, "Project: MyCoolProject"))
		assert.True(t, strings.Contains(job.Body, "010101"))
		assert.True(t, strings.Contains(job.Body, "Time"))
		assert.True(t, strings.Contains(job.Body, "Place"))
	})
}
