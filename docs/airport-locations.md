---
title: Airport locations
---

# Airport Locations

Airtrack can locate the takeoff and landing airport of a flight. In order to do this,
it needs to be configured with a source of airport information. Currently it supports
loading airport locations from files in openaip `aip` format[^1], or SeeYou `cup` format[^2].

## openAIP

[openAIP](http://openaip.net) is an open, community-driven source for airport information. The
database is available under open source [CC BY-NC-SA 3.0 license](https://creativecommons.org/licenses/by-nc-sa/3.0/).

Airtrack releases come with the latest available copy built in. If you notice something missing
from your countries airport file, please add it on openAIP so they get included in a later release.

## Custom airport lists

If you know of airport locations and want them to work with airtrack, consider writing your own `aip`
or `cup` file. The [airport configuration section](configuration.html#airports_config) allows you
to specify your own airport files.

## References

[^1]: [openAIP AIP file format V1.1](http://www.openaip.net/system/files/openAIP_aip_format_1_1_airport.pdf)

[^2]: [SeeYou CUP file format - 2018-06-07](http://download.naviter.com/docs/CUP-file-format-description.pdf)