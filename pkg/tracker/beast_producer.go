package tracker

import (
	"context"
	"fmt"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/readsb"
	"net"
	"strconv"
	"sync"
	"time"
)

// BeastProducer - implements Producer.
// This type represents a connection to a beast server.
type BeastProducer struct {
	host      string
	port      uint16
	decoder   *readsb.Decoder
	messages  chan *pb.Message
	wg        sync.WaitGroup
	canceller func()
}

// NewBeastProducer initializes a new BeastProducer.
func NewBeastProducer(msgs chan *pb.Message, host string, port uint16) *BeastProducer {
	return &BeastProducer{
		messages: msgs,
		host:     host,
		port:     port,
		decoder:  readsb.NewDecoder(),
	}
}

// Name - see Producer.Name()
func (p *BeastProducer) Name() string {
	return fmt.Sprintf("beast(%s:%d)", p.host, p.port)
}

// Start - see Producer.Start()
// This function starts the producer goroutine, and the readsb
// periodic update goroutine.
func (p *BeastProducer) Start() {
	p.wg.Add(2)
	ctx, canceller := context.WithCancel(context.Background())
	p.canceller = canceller
	go p.trackPeriodicUpdate(ctx)
	go p.producer(ctx)
}

// trackPeriodicUpdate is a goroutine that triggers readsb
// to check for missing aircraft. It terminates when the provided
// context signals done.
func (p *BeastProducer) trackPeriodicUpdate(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-time.After(time.Second * 30):
			readsb.TrackPeriodicUpdate(p.decoder)
		case <-ctx.Done():
			return
		}
	}
}

// producer is a goroutine that connects to the beast server
// and reads it's contents. The decoded stream of messages is
// converted and sent over the messages channel.
func (p *BeastProducer) producer(ctx context.Context) {
	defer p.wg.Done()

	for {
		// Connect to server (with a timeout). If an error is returned,
		// wait 30 seconds and try again.
		conn, err := net.DialTimeout("tcp",
			fmt.Sprintf("%s:%d", p.host, p.port), time.Second*5)
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
			// Stop if we received the close signal.
			select {
			default:
			case <-ctx.Done():
				return
			}

			// Set a read deadline for the next read.
			err = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if err != nil {
				_ = conn.Close()
				// Sleep for 30 seconds and attempt to reconnect, or quit
				// if we received the quit signal.
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
				// Sleep for 30 seconds and attempt to reconnect, or quit
				// if we received the quit signal.
				select {
				case <-time.After(time.Second * 30):
					continue
				case <-ctx.Done():
					return
				}
			}

			// append receivedData to connection buffer for leftovers from
			// last read
			connBuf = append(connBuf, recvBuf...)

			// Parse messages from buffer and send
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
					proto.AltitudeGeometric = strconv.FormatInt(altitude, 10)
				}
				if altitude, err := msg.GetAltitudeBaro(); err == nil {
					proto.AltitudeBarometric = strconv.FormatInt(altitude, 10)
				}
				if rate, err := msg.GetRateGeom(); err == nil {
					proto.HaveVerticalRateGeometric = true
					proto.VerticalRateGeometric = int64(rate)
				}
				if rate, err := msg.GetRateBaro(); err == nil {
					proto.HaveVerticalRateBarometric = true
					proto.VerticalRateBarometric = int64(rate)
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
				if alt, err := msg.GetFmsAltitude(); err == nil {
					proto.HaveFmsAltitude = true
					proto.FmsAltitude = alt
				}
				p.messages <- proto
			}

			// If we did parse anything, move the tail of the message to
			// the start of the connection buffer for next time.
			if som > 0 {
				// slice tail of buffer and set to start
				connBuf = connBuf[som:]
			}
		}
	}
}

// Stop sends the cancel signal to the producer + trackPeriodicUpdate goroutines
// and blocks until they finish processing
func (p *BeastProducer) Stop() {
	p.canceller()
	p.wg.Wait()
}
