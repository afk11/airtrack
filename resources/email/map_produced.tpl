Project: {{.Project}}<br />

{{if .CallSign}}
Map produced for {{.Icao }} {{.CallSign}}.
{{else}}
Map produced for {{.Icao }} flight.
{{end}}
<br />
Duration: {{.DurationFmt}}

<br />
First seen:<br />
<ul>
    <li>Time: {{ .StartTimeFmt }}</li>
    <li>Place: <a href="https://www.openstreetmap.org/#map=13/{{ .StartLocation.Longitude }}/{{ .StartLocation.Longitude }}">{{ .StartLocation.Latitude }}, {{ .StartLocation.Longitude  }}</a> @ {{ .StartLocation.Altitude }} ft</li>
</ul>

Last seen:<br />
<ul>
    <li>Time: {{ .EndTimeFmt }}</li>
    <li>Place: <a href="https://www.openstreetmap.org/#map=13/{{ .EndLocation.Latitude }}/{{ .EndLocation.Longitude }}">{{ .EndLocation.Latitude }}, {{ .EndLocation.Longitude  }}</a> @ {{ .EndLocation.Altitude }} ft</li>
</ul>
