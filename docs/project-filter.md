---
title: Project filter
---

# Project filters

Filters allow us to make our tracking project focused on certain aircraft or conditions.

Filters are simple expressions which are evaluated on incoming messages. If the expression
returns false, the project ignores the message.

Airtrack uses [Googles CEL - Common Expression Language](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
for writing, parsing, and evaluation expressions. For an introduction to CEL, [see this guide](https://github.com/google/cel-spec/blob/master/doc/intro.md)

The expressions have access to the current state of the aircraft, and the incoming
message.
 * `state` - the State message for this aircraft
 * `m` - the new Message

The definition for these structures is here:
```proto
syntax = "proto3";
package airtrack;
option go_package = "github.com/afk11/airtrack/pkg/pb";

message Source {
  string Id = 1;
  string Type = 2;
  string Url = 3;
};
message Message {
  Source Source = 1;
  // 6 character hex identifier for aircraft
  string Icao = 10;
  string Squawk = 11;
  string CallSign = 12;
  string Altitude = 13;

  string Latitude = 20;
  string Longitude = 21;

  bool IsOnGround = 30;

  string VerticalRate = 40;

  string Track = 50;
  string GroundSpeed = 90;
}
message State {
  // 6 character hex identifier for aircraft
  string Icao = 1;

  bool HaveAltitude = 10;
  // Aircraft altitude in feet
  int64 Altitude = 11;

  bool HaveLocation = 20;
  // Latitude
  double Latitude = 21;
  // Longitude
  double Longitude = 22;

  bool HaveCallsign = 30;
  // Callsign or flight identifier
  string CallSign = 31;

  bool HaveSquawk = 40;
  // 4 digit octal number (as string)
  string Squawk = 41;

  bool HaveCountry = 50;
  // Aircraft registration country determined by ICAO Country Allocation
  // CountryCode is ISO3166 2 letter code
  string CountryCode = 51;
  // Country is the long country name
  string Country = 52;

  // IsOnGround tracks whether the aircraft is on ground or in the air.
  bool IsOnGround = 60;

  bool HaveVerticalRate = 70;
  // VerticalRate is the change in vertical rate (+/-) in feet per minute
  int64 VerticalRate = 71;

  bool HaveTrack = 80;
  double Track = 81;

  bool HaveGroundSpeed = 90;
  double GroundSpeed = 91;
}
```

## Example Filters

We can detect the Google WIFI balloons based their altitude. An expression to track only aircraft above 50000ft:

    state.Altitude > 50000

Using the state.CountryCode field, we can focus on aircraft from a certain country:

    state.CountryCode == "US"

Simple expressions above can be combined with the logical AND operator (`&&`) to restrict sightings to aircraft from the United States above 50000ft

    state.CountryCode == "US" && state.Altitude > 50000

If we wanted to include balloons from Russia and Norway we can use the logical OR operator (`||`)

    (state.CountryCode == "US" || state.CountryCode == "RU" ||
         state.CountryCode == "NO") && state.Altitude > 50000
