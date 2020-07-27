# airtrack

The airtrack project provides software to aid in monitoring of aircraft.

 - dump1090 output
 - the adsbexchange api (if you're a feeder or pay for one)
 - Several data sources: Setup your local dump1090 feeder, or the adsbexchange API if you're a feeder
 - Multiple project support: The configuration file lets you specify multiple projects, each with its own configuration for logging/reporting flight information
 - Filtering language: Projects can define filters to only focus on aircraft that meet specified conditions
 - Database storage: Projects save some minimal information about sighting open and close times, and the aircraft ICAO, as well as other data points about the flight.
 - Monitoring features: Features can be configured per-project. See the specific documentation for each.
 - Email notifications: Certain flight events also trigger email notifications - enabled notifications can be specified in the config file
 - Location geocoding: Find the nearest airports by importing OpenAIP files from https://openaip.net

### Configuration

The main configuration file defines the application configuration, such as databases, SMTP support, or other backend related options.

Projects may also be defined in the main configuration file, and additional projects-only configuration files can loaded via CLI flags.

[See the example configuration file](./example.config.main.yml)

### Projects

Projects are used to organize and manage multiple filters + features.

When received, the message will be processed by all projects. Projects
can filter out certain messages and aircraft by setting a filter. Without
a filter, the features are active for all messages and aircraft.

#### Filtering

The project uses [Google's Common Expression Language](https://github.com/google/cel-spec/blob/master/doc/intro.md) to evaluate filter expressions.

Filter expressions evaluate to a boolean (true/false) result, and may operate on fields in the following input variables:
 - state: The current state of the aircraft
 - msg: The current message being processed

[See of filter variables](./pb/message.proto)

Some example expressions:

    # Only process messages from the Antonov-225
    msg.Icao == "508035"

    # Only process messages from aircraft whose ICAO registration is
    # to the USA, and whose altitude is above 60000ft
    (state.Country == "US" && state.Altitude > 60000)

#### Features

 - "track_tx_types": maintain a record of different modes/adsb message types used (if the messages come from a dump1090 instance).
 - "track_callsigns": maintain the current callsign of the aircraft, and all callsigns used throughout the sighting.
 - "track_squawks": maintain the current squawk of the aircraft, and all squawks used throughout the sighting.
 - "track_kml": record locations broadcast by the aircraft, and generate a Google Earth KML file plotting its course + altitude.
 - "track_takeoff": monitor for aircraft in the takeoff state (only in logs currently)
 - "geocode_endpoints": reverse location lookup sighting origin and destination positions (only in logs currently)

#### Email notifications

Certain features can trigger email notifications. Each project defines its own list of email notifications to actually send.

 - "map_produced": Triggered when a sighting closes. The message contains the initial and final time and location, and has the Google Earth KML file attached.
 - "spotted_in_flight": Triggered when a sighting is opened. The message includes the time/location/ICAO/callsign.
 - "takeoff_start": Triggered when a takeoff first begins.
 - "takeoff_complete": Triggered when a takeoff is complete.

## Building the software

Install go-bindata, and proto-gen-go

    make build-airtrack-linux-amd64

## Run the software

A valid configuration file is required to run airtrack.

### Database migrations

After copying and modifying [the example configuration file](./example.config.main.yml) to meet your needs,
the database migrations need to be run.

    Usage: airtrack.linux-amd64 <command>

    Flags:
      --help                     Show context-sensitive help.
      --config=STRING            Configuration file path
      --projects=PROJECTS,...    Projects configuration file (may be repeated, and in addition to main
                                 configuration file)

    Commands:
      track
        Track aircraft

      migrate up
        Migrate to latest database migration

      migrate down
        Rollback all migrations

      migrate steps
        Migrate n steps forward if positive, or rollback n if negative

    Run "airtrack.linux-amd64 <command> --help" for more information on a command.


## Contribute

## Resources

CUP File Format

http://download.naviter.com/docs/CUP-file-format-description.pdf

OpenAIP Airport File Format

http://www.openaip.net/system/files/openAIP_aip_format_1_1_airport.pdf