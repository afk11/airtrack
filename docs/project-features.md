---
title: Project tracking features
---

# Tracking Features

A tracking feature can be enabled causing extra flight information to be recorded in the database.

The list of supported features:
 * [track_tx_types](#track_tx_types)
 * [track_callsigns](#track_callsigns)
 * [track_squawks](#track_squawks)
 * [track_kml](#track_kml)
 * [track_takeoff](#track_takeoff)
 * [geocode_endpoints](#geocode_endpoints)

## track_tx_types

`track_tx_types` can be enabled to survey the ADS-B message types transmitted by an aircraft
during the flight.

The record of observed transmission types is stored as a uint8 on the `sighting` record.
Each bit position corresponds to a certain message type - the bit being set to 1 indicates
that this message type was observed.

**Note** this feature only works on messages received from a dump1090 instance.

| Bit | Message Type | Message description                                                    |
| --- | ------------ | :--------------------------------------------------------------------- |
| 0   | DF17 BDS 0,8 | ES Identification and Category                                         |
| 1   | DF17 BDS 0,6 | ES Surface Position Message (Triggered by nose gear squat switch.)     |
| 2   | DF17 BDS 0,5 | ES Airborne Position Message                                           |
| 3   | DF17 BDS 0,9 | ES Airborne Velocity Message                                           |
| 4   | DF4, DF20    | Surveillance Alt Message (Triggered by ground radar. Not CRC secured.) |
| 5   | DF5, DF21    | Surveillance ID Message (Triggered by ground radar. Not CRC secured.)  |
| 6   | DF16         | Air To Air Message (Triggered from TCAS)                               |
| 7   | DF11         | All Call Reply (Broadcast but also triggered by ground radar)          |

## track_callsigns

`track_callsigns` controls whether the callsigns used by aircraft should be recorded.

When enabled, the `sighting.callsign` record will contain the latest callsign used by the aircraft.
`sighting_callsign` records are created every time the callsign changes, forming a journal of
all callsigns used and when the callsign was changed.

## track_squawks

`track_squawks` controls whether the squawks used by aircraft should be recorded.

When enabled, the `sighting.squawk` record will contain the latest squawk used by the aircraft.
`sighting_squawk` records are created every time the squawk is changed, forming a journal of
all squawks used and when the squawk was changed.

## track_kml

The `track_kml` feature makes the project record each position broadcast by the aircraft. Once
the sighting is closed, the journal of locations is processed into a KML file showing the movements
of the aircraft. If the `reopen_sightings` is `true`, and a processed sighting is reopened, the
 location log will be updated once again, and a new KML produced when the sighting closes.

Upon each position update (altitude, latitude, longitude), a new log is added to the `sighting_location`
table. The KML file is kept in `sighting_kml` and is associated with the `sighting`.

**Note** this feature is required for [map_produced](project-event-notifications.html#map_produced)
notifications to be produced.

## track_takeoff

monitor for aircraft in the takeoff state (only in logs currently)

## geocode_endpoints
reverse location lookup sighting origin and destination positions (only in logs currently)
