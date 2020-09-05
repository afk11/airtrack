package tracker

type AdsbxAircraftResponse struct {
	Aircraft []AdsbxAircraft `json:"ac"`
	Msg      string          `json:"msg"`
	Total    int64           `json:"total"`
	CTime    int64           `json:"ctime"`
	PTime    int64           `json:"ptime"`
}

type AdsbxAircraft struct {
	PosTime      string `json:"postime"`
	Icao         string `json:"icao"`
	Registration string `json:"reg"`
	Type         string `json:"type"`
	Wtc          string `json:"wtc"`
	Spd          string `json:"spd"`
	Altt         string `json:"altt"`
	Alt          string `json:"alt"`
	Galt         string `json:"galt"`
	Talt         string `json:"talt"`
	Lat          string `json:"lat"`
	Lon          string `json:"lon"`
	Vsit         string `json:"vsit"`
	Vsi          string `json:"vsi"`
	Trkh         string `json:"trkh"`
	Ttrk         string `json:"ttrk"`
	Trak         string `json:"trak"`
	Sqk          string `json:"sqk"`
	Call         string `json:"call"`
	Ground       string `json:"gnd"`
	Trt          string `json:"trt"`
	Pos          string `json:"pos"`
	Mlat         string `json:"mlat"`
	Tisb         string `json:"tisb"`
	Sat          string `json:"sat"`
	Opicao       string `json:"opicao"`
	Country      string `json:"cou"`
}
