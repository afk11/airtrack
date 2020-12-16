# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2020-10-09

### Added

 - Adds a changelog to the project.

## [0.0.2] - 2020-10-11

### Added

 - Now automatically building releases for linux-amd64

## [0.0.3] - 2020-10-11

### Added

 - Include openAIP data with distributed releases
 - Add `version` command to airtrack

## [0.0.4] - 2020-12-15

### Added

 - Adds `location_update_interval` option to project configuration. Enables frequency
   of location updates to be managed to one per interval
 - Add `location_update_interval` option to global sightings configuration. Allows
   control of the default `location_update_interval` if a project has none configured.
 - Adds a `Signal` proto message, which contains RSSI signal information about
   a message.
 - `Message` proto message has new field `Signal` of type `Signal`. Since its
   an object type, filters can test whether the field is set with `has()`
 - `State` proto message has new field `LastSignal` of type `Signal`. It contains
   the `Signal` value of the most recent message
 - `AircraftMap` now includes RSSI in the aircraft.json
 - `State` and `Message` are updated with new fields: `IndicatedAirSpeed`, `TrueAirSpeed`,
   `Mach`, `Roll`, `NavHeading`, `NavRoll`, `ADSBVersion`, `NACP`, `NACV`, `NICBaro`,
   `SIL`, `SILType`

### Changed

 - Fixes BEAST message processing - call `TrackUpdateFromMessage` ASAP so our message
   is updated with the processed information.
 - Fixes deadlock in tracker/map.go:updateJSON. Release projMu ASAP since ProjectHistory
   also needs it. Debugged using go-deadlock.
 - BEAST Server configuration: default to 30005 if nothing provided