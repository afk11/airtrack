---
title: Tracking lifecycle
---

# Tracking lifecycle

This document describes the sequence of events that take place while an aircraft is tracked.

## New and updated sightings

Airtrack maintains basic information in memory about all aircraft it receives messages about.
This is required as filters need up to date information to perform their function.

This section mostly involves maintaining this information.

When a message is received from an aircraft which was not in view before the following steps are taken:
 1. Create an in-memory `Sighting` object referencing the aircrafts hex ICAO. This object is in memory,
    and contains a `State` proto message, and a list of `ProjectObservations` for each project following
    this aircraft. The `ProjectObservation` structure contains per-project information about the sighting.

 2. Update the aircraft `State` object with information from the current message. This means that airtrack
    contains some minimal information about every aircraft in view. However, information is stored only if
    there is at least one project following the aircraft.

## Project tracking

Airtrack has the project concept to help organize aircraft tracking. We might have multiple projects because
we are interested in different things, or want some behaviours enabled for certain aircraft but not others.

A project may define a filter to focus it's tracking to certain aircraft.
If a filter is set, it will be evaluated. If the expression returns `true`, the project will begin to track - or continue to track - the
aircraft (otherwise, the message will be ignored).

If no project filter is set, the project will track every aircraft.

This section describes the action taken by a project with no filter, or a project whose filter evaluated
to `true`.

 1. Ensure the `Sighting` object contains the `aircraft` database record. If it's not set yet (the first
    time a project has worked on this `Sighting`), we search for the `aircraft` by it's hex ICAO, or insert
    the `aircraft` record if it didn't exist.

 2. If the `Sighting` object doesn't have a `ProjectObservation` for our project yet (the first time this
    project has worked on this `Sighting`), we create a `sighting` record in the database associating our
    project with the `aircraft` record. The `sighting` record is attached to the `ProjectObservation`
    allowing the project to associate tracking information with it's `sighting` in the database. This is
    referred to opening a sighting.

 3. Compare our `ProjectObservation` copy of data with the `State` object, to see if any data has been
    changed. In this phase, we check for enabled features, and save any relevant flight information to our
    database. It can also result in some event notifications being triggered. If so, and the project
    has subscribed to these events, an email job is created and queued for delivery. Documentation for each
    `feature` [can be found here](project-features.html), and documentation for `event_notifications`
    [can be found here](project-event-notifications.html)

## Closing a sighting

The tracker looks for aircraft which haven't sent any messages for a configurable timeout.

This may be due to bad ADS-B coverage, or because the aircraft has landed and powered off. The
resulting action (invoking `handleLostAircraft`) is also triggered when the software receives the
shutdown signal.

Every 5 seconds, airtrack will inspect each `Sighting` and it's `ProjectObservation` list.

If the `Sighting` has some subscribed projects, we check if `sighting_timeout` has elapsed since
the `ProjectObservation` last message time for each project. If so, the projects sighting will
be closed, invoking `handleLostAircraft`.
If there are no subscribed projects, we check if `sighting_timeout` has elapsed since the
`Sighting` last message time. If so, they are deleted from the internal `Sighting` map.