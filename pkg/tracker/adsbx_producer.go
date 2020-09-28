package tracker

import (
	"context"
	"fmt"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	// DefaultAdsbxEndpoint - the default URL to use
	DefaultAdsbxEndpoint = "https://adsbexchange.com/api/aircraft/json/"
)

// jsonDecodeError is a custom error type used to indicate
// that json parsing failed. It contains the decoding error,
// and the response data which lead to the error.
type jsonDecodeError struct {
	err      error
	response []byte
}

// Returns the response data.
func (e *jsonDecodeError) Response() []byte {
	return e.response
}

// Return the internal error message.
func (e *jsonDecodeError) Error() string {
	return e.err.Error()
}

// AdsbxProducer. See Producer.
// This type is responsible for polling the ADSB Exchange API using
// the provided url and apikey. The JSON result is parsed and into messages
// which are written to the messages channel.
type AdsbxProducer struct {
	url                 string
	apikey              string
	messages            chan *pb.Message
	wg                  sync.WaitGroup
	numReqs             int
	jsonPayloadDumpFile string
	canceller           func()
}

// NewAdsbxProducer returns an AdsbxProducer.
func NewAdsbxProducer(msgs chan *pb.Message, url string, apikey string) *AdsbxProducer {
	return &AdsbxProducer{
		messages:            msgs,
		jsonPayloadDumpFile: "/tmp/airtrack-adsbx-json-payload",
		url:                 url,
		apikey:              apikey,
	}
}

// GetAdsbx performs a HTTP request to ADSB Exchange and sends
// messages over the msgs channel.
func (p *AdsbxProducer) GetAdsbx(client *http.Client, ctx context.Context, msgs chan *pb.Message, source *pb.Source) error {
	start := time.Now()
	cancelled := make(chan bool)
	p.numReqs++
	numReq := p.numReqs
	defer func() {
		cancelled <- true
	}()

	go func() {
		select {
		case <-time.After(time.Minute):
			panic(fmt.Errorf("adsbx request (%d) running after 1 minute", numReq))
		case <-cancelled:
			log.Debugf("adsbx request (%d) terminated normally after %s", numReq, time.Since(start))
			break
		}
	}()

	msg := &AdsbxAircraftResponse{}
	var body []byte
	var err error
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
		return errors.Errorf("adsbx request (%d) received not-ok code %d", numReq, resp.StatusCode)
	}

	// check status
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = easyjson.Unmarshal(body, msg)
	if err != nil {
		return &jsonDecodeError{err, body}
	}

	for _, ac := range msg.Aircraft {
		msg := &pb.Message{
			Source:             source,
			Icao:               ac.Icao,
			Squawk:             ac.Sqk,
			CallSign:           ac.Call,
			AltitudeBarometric: ac.Alt,
			AltitudeGeometric:  ac.Galt,
			Latitude:           ac.Lat,
			Longitude:          ac.Lon,
			IsOnGround:         ac.Ground == "1",
			Track:              ac.Trak,
			GroundSpeed:        ac.Spd,
		}
		if ac.Vsi != "" {
			vsi, err := strconv.ParseInt(ac.Vsi, 10, 64)
			if err != nil {
				return err
			}
			if ac.Vsit == "0" {
				msg.HaveVerticalRateBarometric = true
				msg.VerticalRateBarometric = vsi
			} else if ac.Vsit == "1" {
				msg.HaveVerticalRateBarometric = true
				msg.VerticalRateBarometric = vsi
			}
			// any other value for Vsit is unexpected
		}
		if ac.Talt != "" {
			fmsAlt, err := strconv.ParseInt(ac.Talt, 10, 64)
			if err != nil {
				return err
			}
			msg.HaveFmsAltitude = true
			msg.FmsAltitude = fmsAlt
		}

		msgs <- msg
	}

	return nil
}

// producer is a goroutine which periodically calls GetAdsbx to
// receive messages from ADSB Exchange. It terminates if the
// stop signal is received from the provided context.
func (p *AdsbxProducer) producer(ctx context.Context) {
	defer p.wg.Done()

	normalWait := time.Second * 2
	wait := normalWait
	src := &pb.Source{
		Type: pb.Source_AdsbExchange,
		Name: "adsbx",
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
						log.Warnf("adsbx request (%d) had invalid response JSON. writing payload to %s", p.numReqs, p.jsonPayloadDumpFile)
						_ = ioutil.WriteFile(p.jsonPayloadDumpFile, jsonErr.response, 0644)
					} else {
						log.Warnf("adsbx request (%d) had invalid response JSON.", p.numReqs)
					}
				} else {
					log.Warnf("adsbx request (%d) error: %s", p.numReqs, err.Error())
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

// Name - returns the name for this producer. See Producer.Name()
func (p *AdsbxProducer) Name() string {
	return "adsbx"
}

// Start starts the producer goroutine, and the readsb
// periodic update goroutine. See Producer.Start()
func (p *AdsbxProducer) Start() {
	p.wg.Add(1)
	ctx, canceller := context.WithCancel(context.Background())
	p.canceller = canceller
	go p.producer(ctx)
}

// Stop sends the cancel signal to the producer goroutine
// and blocks until it finishes. See Producer.Stop()
func (p *AdsbxProducer) Stop() {
	p.canceller()
	p.wg.Wait()
}
