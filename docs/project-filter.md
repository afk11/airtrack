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

If we wanted to track all United Airlines flights, we can use the `state.OperatorCode` field
derived from the callsign: 
 
     state.OperatorCode == "UAL"

Some State and Message properties are objects. For instance, the `State` objects
`Info` property is a structure of type `AircraftInfo`. This property is only set if the
information can be found in the database. The function `has()` returns whether the property
is available.

The following filter ensures only Eurocopter EC135 aircraft will be tracked:

    has(state.Info) && state.Info.Type == "EC35"

To only process aircraft messages from ADSB Exchange:

    msg.Source.Type == AdsbExchangeSource

To only process messages from a home BEAST server:

    msg.Source.Type == BeastSource && msg.Source.Name == "home"

# Constants

The following constants can be used in filter expressions:

 - Type: `Source.SourceType`.
   - `AdsbExchangeSource`: message source was ADSB Exchange
   - `BeastSource`: message source was a BEAST server

# Definitions

The definition for these structures is here:
```proto
syntax = "proto3";
package airtrack;
option go_package = "github.com/afk11/airtrack/pkg/pb";

// Source contains information about which receiver produced the message
message Source {
  // SourceType - enumeration of types of message producers
  enum SourceType {
    AdsbExchange = 0;
    BeastServer = 1;
  }
  // Name - name of the producer. ADSB Exchange is 'adsbx'.
  // Beast Servers use the name from the config entry.
  string Name = 1;
  // Type - type of producer that produced this message
  SourceType Type = 2;
};
// Signal contains signal strength information about the received message
message Signal {
  // Rssi - signal strength
  double Rssi = 1;
};
// AircraftInfo represents an entry in the readsb database, containing
// information about the aircraft
message AircraftInfo {
  // Registration - assigned registration for the aircraft.
  string Registration = 1;
  // TypeCode - identifies the type of aircraft
  string TypeCode = 2;
  // F
  string F = 3;
  // Description - brief description of the aircraft type for humans
  string Description = 4;
}
// Operator represents an entry in the readsb operators database, and
// contains information about the operator of the current flight.
message Operator {
  // Name - name of the operator
  string Name = 1;
  // CountryName - where the operator is based
  string CountryName = 2;
  // R
  string R = 3;
}
// Message - a payload produced by one of our receivers
message Message {
  // Source identifies the receiver which produced the message
  Source Source = 1;
  // Signal contains information about the signal strength. Only
  // set for BEAST messages currently.
  Signal Signal = 2;

  // Icao - 6 character hex identifier for aircraft
  string Icao = 10;
  // Squawk - a 4 digit octal squawk code (as a string)
  string Squawk = 11;
  // CallSign - aircrafts flight ID/callsign
  string CallSign = 12;
  // AltitudeGeometric - geometric altitude
  string AltitudeGeometric = 13;
  // AltitudeBarometric - barometric altitude
  string AltitudeBarometric = 14;

  // Latitude - latitude coordinate
  string Latitude = 20;
  // Longitude - longitude coordinate
  string Longitude = 21;

  // IsOnGround is '1' if the aircraft is on ground, '0' otherwise
  bool IsOnGround = 30;

  // VerticalRateGeometric - change in altitude by ft per minute (UNITS??)
  int64 VerticalRateGeometric = 40;
  // HaveVerticalRateGeometric - whether VerticalRateGeometric is set
  bool HaveVerticalRateGeometric = 41;

  // VerticalRateBarometric - change in altitude by ft per minute (UNITS??)
  int64 VerticalRateBarometric = 45;
  // HaveVerticalRateBarometric - whether VerticalRateBarometric is set
  bool HaveVerticalRateBarometric = 46;

  string Track = 50;
  double MagneticHeading = 51;
  bool HaveMagneticHeading = 52;
  double TrueHeading = 53;
  bool HaveTrueHeading = 54;

  // HaveFmsAltitude - used to indicate that FmsAltitude is set
  bool HaveFmsAltitude = 60;
  // FmsAltitude - the target altitude set on the aircrafts navigation
  int64 FmsAltitude = 61;

  // HaveNavHeading - used to indicate that NavHeading is set
  bool HaveNavHeading = 65;
  // NavHeading - heading set in navigation
  double NavHeading = 66;

  // HaveCategory - used to indicate that Category is set
  bool HaveCategory = 70;
  // Category - type of the transponder
  string Category = 71;

  // GroundSpeed - speed in knots
  string GroundSpeed = 90;

  // HaveTrueAirSpeed indicates whether TrueAirSpeed is set
  bool HaveTrueAirSpeed = 91;
  // TrueAirSpeed is the true airspeed in knots
  // todo: units?
  uint64 TrueAirSpeed = 92;
}

// State contains general information about a sighting.
message State {
  // 6 character hex identifier for aircraft
  string Icao = 1;
  // Contains aircraft registration, type, and description of the aircraft
  AircraftInfo Info = 2;
  // OperatorCode is a three letter code which references the operator for
  // this flight. The field is not empty if the callsign begins with three
  // letters followed by a number.
  string OperatorCode = 3;
  // Operator information contains the name of the operator and its country.
  // It will only be set if the `OperatorCode` is found in the database.
  Operator Operator = 4;

  // LastSignal contains the signal strength from the last message.
  Signal LastSignal = 5;

  // barometric altitude in feet
  bool HaveAltitudeBarometric = 10;
  int64 AltitudeBarometric = 11;

  // geometric altitude in feet
  bool HaveAltitudeGeometric = 12;
  int64 AltitudeGeometric = 13;

  // HaveLocation - used to indicate whether Latitude and Longitude are set.
  bool HaveLocation = 20;
  // Latitude
  double Latitude = 21;
  // Longitude
  double Longitude = 22;

  // HaveCallsign - indicates whether Callsign is set.
  bool HaveCallsign = 30;
  // Callsign or flight identifier
  string CallSign = 31;

  // HaveSquawk - indicates whether Squawk is set.
  bool HaveSquawk = 40;
  // Squawk - 4 digit octal number (as string)
  string Squawk = 41;

  // HaveCountry - indicates whether Country and CountryCode fields are set.
  bool HaveCountry = 50;
  // CountryCode - Aircraft registration country determined by ICAO Country Allocation
  // CountryCode is ISO3166 2 letter code
  string CountryCode = 51;
  // Country is the long country name
  string Country = 52;

  // IsOnGround tracks whether the aircraft is on ground or in the air.
  bool IsOnGround = 60;

  // HaveVerticalRateBarometric indicates whether VerticalRateBarometric is set.
  bool HaveVerticalRateBarometric = 70;
  // VerticalRateBarometric - barometric change in vertical rate (+/-) in feet per minute
  int64 VerticalRateBarometric = 71;

  // HaveVerticalRateGeometric indicates whether VerticalRateGeometric is set.
  bool HaveVerticalRateGeometric = 75;
  // VerticalRateGeometric - geometric change in vertical rate (+/-) in feet per minute
  int64 VerticalRateGeometric = 76;

  // HaveTrack indicates whether Track is set.
  bool HaveTrack = 80;
  double Track = 81;

  // HaveFmsAltitude indicates whether FmsAltitude is set.
  bool HaveFmsAltitude = 85;
  // FmsAltitude - the target altitude set on navigation instruments
  int64 FmsAltitude = 86;

  // HaveNavHeading indicates whether NavHeading is set.
  bool HaveNavHeading = 87;
  // NavHeading - the heading set in navigation
  double NavHeading = 88;

  // HaveGroundSpeed indicates whether GroundSpeed is set.
  bool HaveGroundSpeed = 90;
  // GroundSpeed - the ground speed in knots
  double GroundSpeed = 91;
  // HaveTrueAirSpeed indicates whether TrueAirSpeed is set
  bool HaveTrueAirSpeed = 92;
  // TrueAirSpeed is the true airspeed in knots
  // todo: units?
  uint64 TrueAirSpeed = 93;

  // HaveCategory indicates whether Category is set.
  bool HaveCategory = 100;
  // Category - the transponder type
  string Category = 101;
}
```
