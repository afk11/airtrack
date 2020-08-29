---
title: Airport locations
---

# Airport Locations

Airtrack can locate the takeoff and landing airport of a flight. In order to do this,
it needs to be configured with a source of airport information. Currently it supports
loading airport locations from files in openaip `aip` format[^1], or SeeYou `cup` format[^2].

## openAIP

A community backed source for airport information is [openAIP](http://openaip.net). The files
can be downloaded from the `Airport Files` section at [this location](http://www.openaip.net/downloads)

openAIP is a great resource as they have files for most countries. If you notice something missing
from your countries airport file, please add it on openAIP so it can be added to the files.

## Custom airport lists

If you know of airport locations and want them to work with airtrack, consider writing your own `aip`
or `cup` file.

## References

[^1]: [openAIP AIP file format V1.1](http://www.openaip.net/system/files/openAIP_aip_format_1_1_airport.pdf)

[^2]: [SeeYou CUP file format - 2018-06-07](http://download.naviter.com/docs/CUP-file-format-description.pdf)