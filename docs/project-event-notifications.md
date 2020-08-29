---
title: Project event notifications
---

Each project may define a list of events which if they arise will trigger an email notification.

The list of supported events and their description:

<dl>
<dt>takeoff_start</dt>
<dd>Triggered when a takeoff first begins.</dd>
<dt>takeoff_complete</dt>
<dd>Triggered when a takeoff is complete.</dd>
<dt>spotted_in_flight</dt>
<dd>Triggered when a sighting is opened. The message includes the time/location/ICAO/callsign.</dd>
<dt>map_produced</dt>
<dd>Triggered when a sighting closes if a map was created. The message contains the initial and final time and location, and has the Google Earth KML file attached.</dd>
</dl>
