---
title: airtrack - home aircraft monitoring
---

The airtrack project provides software to aid in monitoring of aircraft.

The idea is to [configure](configuration.html) airtrack with projects to
organize tracking interesting aircraft and conditions. Projects store their
sightings and tracking information in the database. [Features](project-features.html)
enable us to decide what information to store besides the bare minimum. [Filters](project-filter.html)
are used exclude aircraft which do not meet the criteria.

Processing flight information can lead to [event notifications](project-event-notifications.html). An email notification will be sent if the project is subscribed to an emitted event.

If airtrack is configured with [airport location files](airport-locations.html), it
can geolocate the takeoff and landing airport for a flight.

See [Aircraft Tracking Lifecycle](tracking-lifecycle.html) for a description of the
key stages in flight tracking.

## Run Airtrack

[Click here](./running-airtrack.html) to see how to setup and run airtrack.