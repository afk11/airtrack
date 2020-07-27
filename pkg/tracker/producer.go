package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultAdsbxEndpoint = "https://adsbexchange.com/api/aircraft/json/"
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
var numReqs int

type AdsbxProducer struct {
	url                 string
	apikey              string
	messages            chan *pb.Message
	wg                  sync.WaitGroup
	jsonPayloadDumpFile string
	canceller           func()
}

func NewAdsbxProducer(msgs chan *pb.Message, url string, apikey string) *AdsbxProducer {
	return &AdsbxProducer{
		messages:            msgs,
		jsonPayloadDumpFile: "/tmp/airtrack-adsbx-json-payload",
		url:                 url,
		apikey:              apikey,
	}
}

func (p *AdsbxProducer) GetAdsbx(client *http.Client, ctx context.Context, msgs chan *pb.Message, source *pb.Source) error {
	start := time.Now()
	cancelled := make(chan bool)
	defer func() {
		cancelled <- true
		numReqs++
	}()

	go func() {
		select {
		case <-time.After(time.Minute):
			panic(fmt.Errorf("running after 1 minute, on request %d", numReqs))
		case <-cancelled:
			log.Debugf("terminated normally after %s", time.Since(start))
			break
		}
	}()

	req, err := http.NewRequest("GET", p.url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("api-auth", p.apikey)
	req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("received not-ok code %d from adsbx", resp.StatusCode)
	}

	// check status
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
		msg := &pb.Message{
			Source:       source,
			Icao:         ac.Icao,
			Squawk:       ac.Sqk,
			CallSign:     ac.Call,
			Altitude:     ac.Alt,
			Latitude:     ac.Lat,
			Longitude:    ac.Lon,
			IsOnGround:   ac.Ground == "1",
			VerticalRate: ac.Vsi,
			Track: ac.Trak,
			GroundSpeed: ac.Spd,
		}
		msgs <- msg
	}

	return nil
}

func (p *AdsbxProducer) producer(ctx context.Context) {
	defer p.wg.Done()

	normalWait := time.Second * 2
	wait := normalWait
	src := &pb.Source{
		Id:   "1",
		Type: "adsbx",
		Url:  p.url,
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	var degradedService bool
	var retryCount int

	for {
		select {
		case <-time.After(wait):
			err := p.GetAdsbx(client, ctx, p.messages, src)
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
				}
				if !degradedService {
					degradedService = true
				}
				retryCount++

				wait = time.Duration(10*retryCount) * time.Second
				log.Warnf("adsbexchange producer %d, sleeping %s", retryCount, wait)
				continue
			}

			if degradedService {
				log.Warnf("adsbexchange - normal service restored after %d retries", retryCount)
				degradedService = false
				retryCount = 0
				wait = normalWait
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
