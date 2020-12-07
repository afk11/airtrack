# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
 - Adds 'location_update_interval' option to project configuration. Enables frequency
   of location updates to be managed to one per interval
 - Adds a `Signal` proto message, which contains RSSI signal information about
   a message.
 - `Message` proto message has new field `Signal` of type `Signal`. Since its
   an object type, filters can test whether the field is set with `has()`

### Changed

 - Fixes BEAST message processing - call `TrackUpdateFromMessage` ASAP so our message
   is updated with the processed information.

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
