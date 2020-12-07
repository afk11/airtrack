---
title: Configuration
---

# Configuration

airtrack is configured via command line flags and a configuration file. The command line
flags define some constant system wide options (location of config files, log level),
and the configuration file sets out database connection settings, email settings, project
settings, and so on.

To view the command line options run `./airtrack track --help`

## Configuration files

Airtrack uses [YAML format](https://en.wikipedia.org/wiki/YAML) for its configuration files.

Placeholders for certain types of data: 
 * `<boolean>`: a boolean taking `true` or `false` as a value
 * `<string>`: a string of characters
 * `<host>`: a valid hostname or IP address
 * `<int>`: an integer value
 * `<secret>`: a secret sequence of characters. passwords/apikeys etc
 * `<ip_address>`: an IP address
 * `<http_url>`: a valid URL with an HTTP or HTTPS schema
 * `<dirpath>`: a valid path to a directory on the filesystem (absolute or relative)
 * `<email>`: a valid email address 
 * `<filter>`: a valid CEL expression. See [Project Filters](project-filter.html) for more information.
 * `<notificationevent>`: a notification event supported by airtrack
 * `<feature>`: a tracking feature supported by airtrack
 * `<mapservice>`: a valid map layout supported by airtrack

This document contains overviews of various YML configuration blocks, the types of
each parameter, and the structures representing entire configuration files.

### Main configuration file

The `--config` command line option specifies the main configuration file.
 
The main configuration file has the following structure:
```yaml
# Database configuration
database: <database_config>

# Local time zone
[ timezone: <string> | default = system timezone ]

# Configuration for ADSBExchange API
[ adsbx: <adsbexchange_config> | default = none ]

# Configuration for beast servers
beast:
  [ - <beast_config> | default = none ]

# Import airport locations for flight source + destination geolocation
[ airports: <airports_config> | default = none ]

# Configuration for the email driver
[ email: <email_config> | default = none ]

# Configuration for the HTTP server with maps
[ map: <map_config> | default = none ]

# Configuration for prometheus metrics
[ metrics: <metrics_config> | default = none ]

# Project configurations
projects:
[ - <project_config> | default = none ]
```

### Projects configuration file
It's also possible to pass additional _projects only_ configuration files using the 
`--projects` command line option. The option can be repeated to pass several project
only configuration files.

A project configuration file has the following structure:
```yaml
# Project configurations
projects:
[ - <project_config> | default = none ]
```

### `<adsbexchange_config>`
[ADS-B Exchange](https://adsbexchange.com) is an unfiltered map of data received
by a community of ADS-B receivers around the world. They provide an API which is
free for feeders or can access can be purchased. 

An API key is required for this configuration.

```yaml
# Your API key for connection to ADS-B Exchange
apikey: <secret>
# URL for ADS-B Exchange. Not required unless you have
# a local cache of the API
[ url: <http_url> | default = "https://adsbexchange.com/api/aircraft/json/" ]
```

### `<beast_config>`
BEAST format messages are produced by dump1090 on port 30005 by default.

```yaml
# Name for this data source
name: <string>
# Hostname or IP for server
host: <host>
# Port for connection (probably 30005)
port: <port>
```

### `<airports_config>`

Airtrack can geolocate the takeoff and landing airport for a flight.
Locations are provided to airtrack via `aip` and `cup` files. Airtrack
releases contain the latest available version of the openaip database.

Use of the built-in openaip locations can be disabled in this configuration section.
It's also possible to configure custom directories containing `aip` and `cup` files.

[Click here for information about airport location files](airport-locations.html)

```yaml
# disable_builtin_airports will prevent airtrack from loading
# compiled-in airport files if set to `true`
[ disable_builtin_airports = <boolean> | default = false ]

# List of directories containing aip files:
openaip: 
  [ - <dirpath> | default = none ]
# List of directories containing cup files:
cup:
  [ - <dirpath> | default = none ]
```

### `<email_config>`

The `<email_config>` section contains configuration related to sending email.

The `driver` field is required along with the configuration for that particular driver.
Currently, only the `smtp` driver is supported.

```yaml
# Email driver. Currently only 'smtp' is supported
driver: <string>

# Configuration for 'smtp' email driver
[ smtp: <smtp_config> | default = none ]
```

### `<smtp_config>`

The `<smtp_config>` section configures the SMTP based email driver.

The `sender`, `username`, `password`, `host`, and `port` fields are required.

To use TLS, `tls` must be set to `true`.

For connections without TLS, the default connection strategy will use STARTTLS if the
server advertises it. STARTTLS can be made mandatory by setting `mandatory_starttls`
to `true`. To disable opportunistic STARTTLS, set `nostarttls` to `true`.
 
```yaml
# sender email address
sender: <email>
# SMTP username
username: <string>
# SMTP password
password: <secret>
# The SMTP server's hostname/ip
host: <host>
# The SMTP server port
port: <int>

# Whether to connect using TLS (port 587)
[ tls: <boolean> | default = false ]

# If `mandatory_starttls` set to true, connections 
# to servers which do not advertise STARTTLS support
# will cause an error.
[ mandatory_starttls: <boolean> | default = true ] 
# disables opportunistic encryption of connection with STARTTLS
# if this is set to true
[ nostarttls: <boolean> | default = false ]
```

### `<map_config>`

The `<map_config>` section contains configuration for the map web server.

This configuration section can be omitted, and the server will start with 
the following defaults:
  - the map is accessible via any interface on port 8080
  - history files created every 30 seconds, and 60 history files are kept (30 minutes of history)
  - tar1090 and dump1090 maps available at ./dump1090/$project and ./tar1090/$project respectively.

```yaml
# Interface the HTTP server will listen on
[ interface: <ip_address> | default = "0.0.0.0" ]
# Port for the HTTP server. Default is 8080
[ port: <int> | default = 8080 ]
# Disable map server. Default is false.
[ disabled: <boolean> | default = false ]
# Number of seconds before creating a new history file. Default
# is every 30 seconds.
[ history_interval: <int> | default = 30 ]
# Number of history files to maintain. Default is to
# store 60 files. Defaults should store 30 minutes of history files.
[ history_count: <int> | default = 60 ]
# The map frontends to provide. Essentially different map skins.
services:
[ - <mapservice> | default = "tar1090", "dump1090" ]
```

### `<metrics_config>`

The `<metrics_config>` section contains confirmation for prometheus metrics about the golang internals and airtrack operations.

If the section is missing or `enabled` is false the metrics server will be disabled.

```yaml
# Whether to enable metrics server.
[ enabled: <boolean> | default = false ]
# Interface the metrics HTTP server will listen on.
[ interface: <ip_address> | default = "0.0.0.0" ]
# Port the metrics HTTP server will listen on.
[ port: <int> | default = 9206 ]
```

### `<database_config>`

A `<database_config>` section is required for airtrack to run. The supported engines are:
 * [MySQL](#mysql)
 * [PostgreSQL](#postresql)
 * [SQLite Version 3](#sqlite)  
 
```yaml
# Database driver. Currently supports 'mysql' and 'sqlite3'
driver: <string>
# DB Host - server to connect to if using mysql
host: <host>
# DB Port - server port if using mysql
port: <int>
# DB Username - username for mysql connection
username: <string>
# DB Password - password for mysql connection
password: <secret>
# Selected Database
# if driver is mysql, this is the database on the mysql server
# if driver is sqlite3, this is the filesystem path to the DB
database: <string|filepath>
```

#### SQLite

SQLite is an embedded storage engine that stores its data in a file.

The `database` option defines the location of the database on the filesystem.
The path may be relative or absolute.

A config file extract of SQLite configuration:
```yaml
# ...
driver: sqlite3
database: /home/user/airtrack.sqlite3
# ...
```

#### MySQL

MySQL is a database server that is connected to over the network.  

A config file extract of MySQL configuration:
```yaml
# ...
driver: mysql
host: server.local
port: 3306
username: airtrack
password: password
database: airtrack
# ...
```

#### PostreSQL

PostgreSQL is a database server that is connected to over the network.  

A config file extract of PostgreSQL configuration:
```yaml
# ...
driver: postgres
host: server.local
port: 3306
username: airtrack
password: password
database: airtrack
# ...
```

### `<project_config>`

A project contains information about how aircraft should be tracked, and optionally,
filters to focus the project on certain aircraft.

The projects [`map` field](#project_map_config) contains per-project configuration of 
the map display. Currently, this only allows the project to be opted out from the map.

Projects can enable `<feature>`'s to determine which data about the flight should be
tracked. [Click here for documentation of available features](project-features.html)

A project can filter aircraft from it's tracking functions if a filter is configured.
[Click here for information about project filters](project-filter.html)

If you'd like to reopen an aircraft sighting if it's visible again within
`reopen_sightings_interval` seconds, `reopen_sightings` can be set to `true`.

Sometimes due to reception problems a message may incorrectly indicate an aircraft
is on the ground. The `onground_update_threshold` defines the number of consecutive
messages to wait before accepting a new `onground` value.

```yaml
# A name for the project
name: <label>

# List of features enabled for the project
features:
[ - <feature> | default = none ]

# Configuration of the map for this project
[ map: <project_map_config> | default = none ]

# Disabled can used to prevent a project from running
[ disabled: <boolean> | default = false ]

# Set a filter for this project. Can be omitted if not required.
[ filter: <filter> | default = none ]

# Project specific notification configuration
[ notifications: <notification_config> | default = none ]

# Whether to reopen a sighting if it was closed recently, due
# to bad coverage, etc
[ reopen_sightings: <boolean> | default = false ]

# How long after an aircraft goes out of range
# before we no longer reopen a recently closed session.
# Unit: seconds.
[ reopen_sightings_interval: <int> | default = 5m ]

# Sometimes due to reception problems a message may incorrectly indicate an aircraft
# is on the ground. The `onground_update_threshold` defines the number of consecutive
# messages to wait before accepting a new `onground` value.
[ onground_update_threshold: <int> | default = 6 ]

# Set a minimum interval in seconds between accepting locations. When zero, all locations
# are accepted. Unit: seconds
[ location_update_interval: <int> | default = 0 ]
```

### `<notification_config>`

A projects `<notification_config>` section defines a list of events 
which should result in an email notification, and an email address
to send notifications to.

[Click here for documentation of available email notifications](project-event-notifications.html)
```yaml
# Destination for notifications
email: <email>

# List of enabled email notifications
events:
[ - <notificationevent> | default = none ]
```

### `<project_map_config>`

A projects `<project_map_config>` section contains per-project configuration
related to the web based map interface. 

Currently the only available option is to opt the project out of the map.

```yaml
# Disabled can be used to prevent a projects map being visible on the map.
[ disabled: <boolean> | default = false ]
```