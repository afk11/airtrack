package tracker

import (
	"context"
	"encoding/json"
	"github.com/afk11/airtrack/pkg/pb"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

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

type jsonDecodeError struct {
	err      error
	response []byte
}

func (e *jsonDecodeError) Response() []byte {
	return e.response
}
func (e *jsonDecodeError) Error() string {
	return e.err.Error()
}

var counter int
var firstTime time.Time

func GetAdsbx(client *http.Client, msgs chan *pb.Message, source *pb.Source) error {
	resp, err := client.Get(source.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//if counter == 0 {
	//	firstTime = time.Now()
	//}
	//if counter == 1 {
	//	if time.Since(firstTime) < 3*time.Minute {
	//		return errors.New("some error")
	//	}
	//}
	//counter++

	msg := &AdsbxAircraftResponse{}
	err = json.Unmarshal(body, msg)
	if err != nil {
		return &jsonDecodeError{err, body}
	}

	for _, ac := range msg.Aircraft {
		msgs <- &pb.Message{
			Source:       source,
			Icao:         ac.Icao,
			Squawk:       ac.Sqk,
			CallSign:     ac.Call,
			Altitude:     ac.Alt,
			Latitude:     ac.Lat,
			Longitude:    ac.Lon,
			IsOnGround:   ac.Ground == "1",
			VerticalRate: ac.Vsi,
		}
	}

	return nil
}

type AdsbxProducer struct {
	messages            chan *pb.Message
	wg                  sync.WaitGroup
	jsonPayloadDumpFile string
	canceller           func()
}

func NewAdsbxProducer(msgs chan *pb.Message) *AdsbxProducer {
	return &AdsbxProducer{
		messages:            msgs,
		jsonPayloadDumpFile: "/tmp/airtrack-adsbx-json-payload",
	}
}

func (p *AdsbxProducer) producer(ctx context.Context) {
	defer p.wg.Done()

	wait := time.Second * 2
	src := &pb.Source{
		Id:   "1",
		Type: "adsbx",
		Url:  "http://sky.oxolan:8080/api/aircraft/json/",
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	var degradedService bool
	var retryCount int

	for {
		select {
		case <-time.After(wait):
			err := GetAdsbx(client, p.messages, src)
			if err != nil {
				jsonErr, ok := (err).(*jsonDecodeError)
				if ok {
					if p.jsonPayloadDumpFile != "" {
						log.Warnf("failed to decode adsbexchange response JSON. writing payload to %s", p.jsonPayloadDumpFile)
						_ = ioutil.WriteFile(p.jsonPayloadDumpFile, jsonErr.response, 0644)
					} else {
						log.Warnf("failed to decode adsbexchange response JSON.")
					}
				} else {
					log.Warnf("unexpected error: %s", err.Error())
					log.Warnf("error: %+v", err)
				}
				if !degradedService {
					degradedService = true
				}
				retryCount++

				sleep := time.Duration(10*retryCount) * time.Second
				log.Warnf("adsbexchange producer %d, sleeping %s", retryCount, sleep)
				time.Sleep(sleep)
				continue
			}

			if degradedService {
				log.Warnf("adsbexchange - normal service restored after %d retries", retryCount)
				degradedService = false
				retryCount = 0
			}
		case <-ctx.Done():
			return
		}
	}
}
func (p *AdsbxProducer) Start() {
	p.wg.Add(1)
	ctx, canceller := context.WithCancel(context.Background())
	p.canceller = canceller
	go p.producer(ctx)
}
func (p *AdsbxProducer) Stop() {
	println("producer issue canceller")
	p.canceller()
	println("producer wait")
	p.wg.Wait()
	println("producer close messages")
	close(p.messages)
}
