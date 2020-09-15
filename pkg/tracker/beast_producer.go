package tracker

import (
	"context"
	"fmt"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/readsb"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"sync"
	"time"
)

type BeastProducer struct {
	host      string
	port      uint16
	decoder   *readsb.Decoder
	messages  chan *pb.Message
	wg        sync.WaitGroup
	canceller func()
}

func NewBeastProducer(msgs chan *pb.Message, host string, port uint16) *BeastProducer {
	return &BeastProducer{
		messages: msgs,
		host:     host,
		port:     port,
		decoder:  readsb.NewDecoder(),
	}
}
func (p *BeastProducer) Name() string {
	return fmt.Sprintf("beast(%s:%d)", p.host, p.port)
}
func (p *BeastProducer) Start() {
	p.wg.Add(1)
	ctx, canceller := context.WithCancel(context.Background())
	p.canceller = canceller
	go p.trackPeriodicUpdate(ctx)
	go p.producer(ctx)
}
func (p *BeastProducer) trackPeriodicUpdate(ctx context.Context) {
	for {
		select {
		case <-time.After(time.Second * 30):
			readsb.TrackPeriodicUpdate(p.decoder)
		case <-ctx.Done():
			return
		}
	}
}

func (p *BeastProducer) producer(ctx context.Context) {
	defer p.wg.Done()

	for {
		str := fmt.Sprintf("%s:%d", p.host, p.port)
		conn, err := net.DialTimeout("tcp", str, time.Second*5)
		if err != nil {
			select {
			case <-time.After(time.Second * 30):
				continue
			case <-ctx.Done():
				return
			}
		}

		connBuf := make([]byte, 2048)
		for {
			select {
			default:
			case <-ctx.Done():
				return
			}

			// set SetReadDeadline
			err = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if err != nil {
				log.Println("SetReadDeadline failed:", err)
				_ = conn.Close()
				// do something else, for example create new conn
				select {
				case <-time.After(time.Second * 30):
					continue
				case <-ctx.Done():
					return
				}
			}

			m := 1024
			recvBuf := make([]byte, m)

			_, err := conn.Read(recvBuf[:]) // recv data
			if err != nil {
				_ = conn.Close()
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					log.Println("read timeout:", err)
					select {
					case <-time.After(time.Second * 30):
						continue
					case <-ctx.Done():
						return
					}
					// time out
				} else {
					log.Println("read error:", err)
					select {
					case <-time.After(time.Second * 30):
						continue
					case <-ctx.Done():
						return
					}
					// some error else, do something else, for example create new conn
				}
			}
			connBuf = append(connBuf, recvBuf...)
			msgs, som, err := readsb.ParseMessage(p.decoder, connBuf)
			if err != nil {
				panic(err)
			}

			for _, msg := range msgs {
				readsb.TrackUpdateFromMessage(p.decoder, msg)

				proto := &pb.Message{
					Icao: msg.GetIcaoHex(),
				}
				if squawk, err := msg.GetSquawk(); err == nil {
					proto.Squawk = squawk
				}
				if callsign, err := msg.GetCallsign(); err == nil {
					proto.CallSign = callsign
				}
				if altitude, err := msg.GetAltitudeGeom(); err == nil {
					proto.Altitude = strconv.FormatInt(altitude, 10)
				} else if altitude, err := msg.GetAltitudeBaro(); err == nil {
					proto.Altitude = strconv.FormatInt(altitude, 10)
				}
				if rate, err := msg.GetRateGeom(); err == nil {
					proto.VerticalRate = strconv.Itoa(rate)
				} else if rate, err := msg.GetRateBaro(); err == nil {
					proto.VerticalRate = strconv.Itoa(rate)
				}
				if heading, err := msg.GetHeading(); err == nil {
					proto.Track = strconv.FormatFloat(heading, 'f', 6, 64)
				}
				if gs, err := msg.GetGroundSpeed(); err == nil {
					proto.GroundSpeed = strconv.FormatFloat(gs, 'f', 1, 64)
				}
				if onground, err := msg.IsOnGround(); err == nil {
					if onground {
						proto.GroundSpeed = "1"
					} else {
						proto.GroundSpeed = "0"
					}
				}
				if lat, lon, err := msg.GetDecodeLocation(); err == nil {
					proto.Latitude = strconv.FormatFloat(lat, 'f', 8, 64)
					proto.Longitude = strconv.FormatFloat(lon, 'f', 8, 64)
				}
				p.messages <- proto
			}
			if som > 0 {
				// slice tail of buffer and set to start
				connBuf = connBuf[som:]
			}
		}
	}
}
func (p *BeastProducer) Stop() {
	p.canceller()
	p.wg.Wait()
}
