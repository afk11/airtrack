package tracker

import (
	"context"
	"fmt"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/readsb"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
)

// BeastProducer - implements Producer.
// This type represents a connection to a beast server.
type BeastProducer struct {
	host      string
	name      string
	port      uint16
	decoder   *readsb.Decoder
	messages  chan *pb.Message
	wg        sync.WaitGroup
	canceller func()
}

// NewBeastProducer initializes a new BeastProducer.
func NewBeastProducer(msgs chan *pb.Message, host string, port uint16, name string) *BeastProducer {
	return &BeastProducer{
		messages: msgs,
		host:     host,
		port:     port,
		name:     name,
		decoder:  readsb.NewDecoder(),
	}
}

// Name - see Producer.Name()
func (p *BeastProducer) Name() string {
	return p.name
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
	source := pb.Source{
		Type: pb.Source_BeastServer,
		Name: p.name,
	}
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
			for i := range msgs {
				msg := msgs[i]
				// call this early so we initialize msg with processed state
				ac := readsb.TrackUpdateFromMessage(p.decoder, msg)
				if ac == nil {
					continue
				}
				recvTime := msg.SysMessageTime()
				proto := &pb.Message{
					Icao:   msg.GetIcaoHex(),
					Source: &source,
				}
				if category, err := ac.GetCategory(); err == nil {
					proto.HaveCategory = true
					proto.Category = category
				} else if category, err := msg.GetCategory(); err == nil {
					proto.HaveCategory = true
					proto.Category = category
				}

				if adsbVersion, err := ac.GetAdsbVersion(); err == nil {
					proto.ADSBVersion = adsbVersion
				}
				if sil, silType, err := ac.GetSIL(recvTime); err == nil {
					proto.HaveSIL = true
					proto.SIL = sil
					proto.SILType = uint32(silType)
				}
				if sil, silType, err := msg.GetSIL(); err == nil {
					proto.HaveSIL = true
					proto.SIL = sil
					proto.SILType = uint32(silType)
				}

				if nacp, err := msg.GetNACP(); err == nil {
					proto.HaveNACP = true
					proto.NACP = nacp
				}
				if nacv, err := msg.GetNACV(); err == nil {
					proto.HaveNACV = true
					proto.NACV = nacv
				}
				if nacv, err := msg.GetNICBaro(); err == nil {
					proto.HaveNICBaro = true
					proto.NICBaro = nacv
				}

				if navModes, err := msg.GetNavModes(); err == nil {
					proto.NavModes = uint32(navModes)
				}
				if qnh, err := msg.GetNavQNH(); err == nil {
					proto.HaveNavQNH = true
					proto.NavQNH = qnh
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
				if heading, headingType, err := msg.GetHeading(); err == nil {
					switch headingType {
					case readsb.HeadingGroundTrack:
						proto.Track = strconv.FormatFloat(heading, 'f', 6, 64)
					case readsb.HeadingMagnetic:
						proto.MagneticHeading = heading
					case readsb.HeadingTrue:
						proto.TrueHeading = heading
					}
				}
				if gs, err := msg.GetGroundSpeed(); err == nil {
					proto.GroundSpeed = strconv.FormatFloat(gs, 'f', 1, 64)
				}
				if alt, err := msg.GetFmsAltitude(); err == nil {
					proto.HaveFmsAltitude = true
					proto.FmsAltitude = alt
				}
				if navHeading, err := msg.GetNavHeading(); err == nil {
					proto.HaveNavHeading = true
					proto.NavHeading = navHeading
				}
				if tas, err := msg.GetTrueAirSpeed(); err == nil {
					proto.HaveTrueAirSpeed = true
					proto.TrueAirSpeed = tas
				}
				if ias, err := msg.GetIndicatedAirSpeed(); err == nil {
					proto.HaveIndicatedAirSpeed = true
					proto.IndicatedAirSpeed = ias
				}
				if mach, err := msg.GetMach(); err == nil {
					proto.HaveMach = true
					proto.Mach = mach
				}
				if roll, err := msg.GetRoll(); err == nil {
					proto.HaveRoll = true
					proto.Roll = roll
				}
				if onground, err := msg.IsOnGround(); err == nil {
					proto.IsOnGround = onground
				}
				if signalLevel, err := msg.GetSignalLevel(); err == nil {
					proto.Signal = &pb.Signal{Rssi: 10 * math.Log10(signalLevel)}
				}
				if lat, lon, err := msg.GetDecodeLocation(); err == nil {
					proto.Latitude = strconv.FormatFloat(lat, 'f', 8, 64)
					proto.Longitude = strconv.FormatFloat(lon, 'f', 8, 64)
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
