---
title: Project tracking features
---

Each aircraft sighting leads to a record in the `sighting` table.

At minimum, the sighting refers to the `aircraft` record and tracks
the sightings `created_at` and `closed_at` time.

The list of supported features and their description:

<dl>
<dt>track_tx_types</dt>
<dd>maintain a record of different modes/adsb message types used (if the messages come from a dump1090 instance).</dd>
<dt>track_callsigns</dt>
<dd>maintain the current callsign of the aircraft, and all callsigns used throughout the sighting.</dd>
<dt>track_squawks</dt>
<dd>maintain the current squawk of the aircraft, and all squawks used throughout the sighting.</dd>
<dt>track_kml</dt>
<dd>record locations broadcast by the aircraft, and generate a Google Earth KML file plotting its course + altitude.</dd>
<dt>track_takeoff</dt>
<dd>monitor for aircraft in the takeoff state (only in logs currently)</dd>
<dt>geocode_endpoints</dt>
<dd>reverse location lookup sighting origin and destination positions (only in logs currently)</dd>
</dl>