---
title: Project event notifications
---

Each project may define a list of events which if they arise will trigger an email notification.

The list of supported events and their description:
 * (takeoff_start)[#takeoff_start]
 * (takeoff_complete)[#takeoff_complete]
 * (spotted_in_flight)[#spotted_in_flight]
 * (map_produced)[#map_produced]

## takeoff_start

This event is triggered when a takeoff first begins. This signal comes from the aircrafts `State.IsOnGround`
field changes from `true` to `false`. Internally, this sets the `IsInTakeoff` `SightingTag`.

## takeoff_unknown_airport

This event is triggered when a takeoff first begins, but we failed to determine the nearest airport.
The trigger and behaviour is the same as `takeoff_start`

## takeoff_complete

This event is triggered when a takeoff is finished. This event can only be produced if the `takeoff_start`
event was encountered, as it relies on the `IsInTakeoff` `SightingTag`. The actual signal comes from
the vertical rate finally being set to zero, if and only if the `IsInTakeoff` `SightingTag` is `true`.

## spotted_in_flight

This event gets triggered when an aircraft sighting is first opened.

## map_produced

This event gets triggered when an aircraft sighting closes, if a KML was produced.

**Note** this event requires the `track_kml` feature to be enabled.