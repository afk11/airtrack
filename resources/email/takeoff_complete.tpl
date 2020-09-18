Project: {{.Project}}<br />

{{.Icao }}
{{if .CallSign}}
 {{.CallSign}}
{{end}}
has completed takeoff {{if .HaveAirport}} from {{.AirportName}} {{end}}
<br />
<br />
<ul>
    <li>Time: {{ .StartTimeFmt }}</li>
    <li>Place: <a href="https://www.openstreetmap.org/#map=13/{{ .StartLocation.Longitude }}/{{ .StartLocation.Longitude }}">{{ .StartLocation.Latitude }}, {{ .StartLocation.Longitude  }}</a> @ {{ .StartLocation.Altitude }} ft</li>
</ul>
