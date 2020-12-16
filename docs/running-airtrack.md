---
title: Running Airtrack
---

# Running Airtrack

This guide explains how setup and run airtrack and view a project's map in the browser.

## Configuration

If you haven't already, [create a configuration file](./configuration.html). Throughout
this guide, we'll assume the main configuration file is called `airtrack.yml` located in
our current working directory.

A database section is mandatory, and you'll need either a local dump1090 instance, or an
ADSB Exchange API key. Use the following configuration to use a local sqlite3 database.
This project has no filter defined, so it will track every aircraft it hears about. It
doesn't enable extra tracking features so shouldn't consume storage too quickly.

```yaml
database:
  driver: sqlite3
  database: airtrack.sqlite3
projects:
  - name: global
```

If you run a dump1090 or readsb instance, you can receive messages from its BEAST output port.
Add the following to your config file:
```yaml
beast:
 - name: home
   host: 10.10.10.92
#   port: 30005   # port is optional, defaults to 30005
```

If you have an ADSB Exchange API key, you can configure that to receive information
about aircraft worldwide from the ADSB Exchange community feeders.
```yaml
adsbx:
  apikey: 42424242-4242-4242-4242-424242424242
```

## Migrations

When setting up airtrack for the first time, we'll need to create a database.

For MySQL and Postgres servers, you'll need to create a new database and user with permissions to
access the same.

For SQLite3, no action is required as the migration tool will create the database file on disk for you.

To run migrations for the first time, or while upgrading from one version to another, the following
command will bring our database schema up to date.

    airtrack migrate up --config=airtrack.yml

## Run Airtrack

Now that we have a configuration file, and a fully initialized database, we can start airtrack
via our shell.

    airtrack track --config=airtrack.yml

Airtrack if there are any problems, airtrack will exit with an error.

## Run Airtrack - systemd

TODO

## Access the map

By default, airtracks map server listens on '0.0.0.0:8080', and enables the map for all projects.
It supports two browser-based map frontends: `dump1090` and `tar1090` both of which enabled by default.

The URL for a map takes the following convention:

    http://localhost:8080/FRONTEND/PROJECT/index.html

For the `tar1090` frontend, the URL for our `global` project [http://localhost:8080/tar1090/global/index.html](http://localhost:8080/tar1090/global/index.html)

For the `dump1090` frontend, the URL is [http://localhost:8080/dump1090/global/index.html](http://localhost:8080/dump1090/global/index.html)

## Reloading configuration

airtrack `track` command responds to the `SIGHUP` signal by closing all sessions, reloading
configuration and starting up again.

## Shutdown

The airtrack `track` command responds to the `SIGTERM` and `SIGINT` by initiating shutdown.