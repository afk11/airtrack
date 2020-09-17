package readsb

// #cgo CFLAGS: -I${SRCDIR}/../../readsb-src/
// #cgo LDFLAGS: -lm -lprotobuf-c ${SRCDIR}/../../readsb-src/readsb.o ${SRCDIR}/../../readsb-src/mode_s.o ${SRCDIR}/../../readsb-src/mode_ac.o ${SRCDIR}/../../readsb-src/ais_charset.o ${SRCDIR}/../../readsb-src/icao_filter.o ${SRCDIR}/../../readsb-src/crc.o ${SRCDIR}/../../readsb-src/comm_b.o ${SRCDIR}/../../readsb-src/util.o ${SRCDIR}/../../readsb-src/cpr.o ${SRCDIR}/../../readsb-src/track.o ${SRCDIR}/../../readsb-src/readsb.pb-c.o ${SRCDIR}/../../readsb-src/geomag.o
// #include <readsb.h>
// #include <station.h>
// #include <track.h>
// #include <mode_s.h>
// #include <mode_ac.h>
// #include <icao_filter.h>
/*
// #include <readsb.h>

// The following functions are helpers because go cannot access C bit fields
int modesmessage_is_altitude_baro_valid(struct modesMessage *mm) {
	return mm->altitude_baro_valid;
}
int modesmessage_is_altitude_geom_valid(struct modesMessage *mm) {
	return mm->altitude_geom_valid;
}
int modesmessage_is_track_valid(struct modesMessage *mm) {
    return mm->track_valid;
}
int modesmessage_is_track_rate_valid(struct modesMessage *mm) {
    return mm->track_rate_valid;
}
int modesmessage_is_heading_valid(struct modesMessage *mm) {
    return mm->heading_valid;
}
int modesmessage_is_roll_valid(struct modesMessage *mm) {
    return mm->roll_valid;
}
int modesmessage_is_gs_valid(struct modesMessage *mm) {
    return mm->gs_valid;
}
int modesmessage_is_ias_valid(struct modesMessage *mm) {
    return mm->ias_valid;
}
int modesmessage_is_tas_valid(struct modesMessage *mm) {
    return mm->tas_valid;
}
int modesmessage_is_mach_valid(struct modesMessage *mm) {
    return mm->mach_valid;
}
int modesmessage_is_baro_rate_valid(struct modesMessage *mm) {
    return mm->baro_rate_valid;
}
int modesmessage_is_geom_rate_valid(struct modesMessage *mm) {
    return mm->geom_rate_valid;
}
int modesmessage_is_squawk_valid(struct modesMessage *mm) {
    return mm->squawk_valid;
}
int modesmessage_is_callsign_valid(struct modesMessage *mm) {
    return mm->callsign_valid;
}
int modesmessage_is_cpr_valid(struct modesMessage *mm) {
    return mm->cpr_valid;
}
int modesmessage_is_cpr_odd(struct modesMessage *mm) {
    return mm->cpr_odd;
}
int modesmessage_is_cpr_decoded(struct modesMessage *mm) {
    return mm->cpr_decoded;
}
int modesmessage_is_cpr_relative(struct modesMessage *mm) {
    return mm->cpr_relative;
}
int modesmessage_is_category_valid(struct modesMessage *mm) {
    return mm->category_valid;
}
int modesmessage_is_geom_delta_valid(struct modesMessage *mm) {
    return mm->geom_delta_valid;
}
int modesmessage_is_from_mlat(struct modesMessage *mm) {
    return mm->from_mlat;
}
int modesmessage_is_from_tisb(struct modesMessage *mm) {
    return mm->from_tisb;
}
int modesmessage_is_spi_valid(struct modesMessage *mm) {
    return mm->spi_valid;
}
int modesmessage_is_spi(struct modesMessage *mm) {
    return mm->spi;
}
int modesmessage_is_alert_valid(struct modesMessage *mm) {
    return mm->alert_valid;
}
int modesmessage_is_alert(struct modesMessage *mm) {
    return mm->alert;
}
int modesmessage_is_emergency_valid(struct modesMessage *mm) {
    return mm->emergency_valid;
}

int modesmessage_is_accuracy_nic_a_valid(struct modesMessage *mm) {
    return mm->accuracy.nic_a_valid;
}
int modesmessage_is_accuracy_nic_b_valid(struct modesMessage *mm) {
    return mm->accuracy.nic_b_valid;
}
int modesmessage_is_accuracy_nic_c_valid(struct modesMessage *mm) {
    return mm->accuracy.nic_c_valid;
}
int modesmessage_is_accuracy_nic_baro_valid(struct modesMessage *mm) {
    return mm->accuracy.nic_baro_valid;
}
int modesmessage_is_accuracy_nac_p_valid(struct modesMessage *mm) {
    return mm->accuracy.nac_p_valid;
}
int modesmessage_is_accuracy_nac_v_valid(struct modesMessage *mm) {
    return mm->accuracy.nac_v_valid;
}
int modesmessage_is_accuracy_gva_valid(struct modesMessage *mm) {
    return mm->accuracy.gva_valid;
}
int modesmessage_is_accuracy_sda_valid(struct modesMessage *mm) {
    return mm->accuracy.sda_valid;
}

int modesmessage_get_accuracy_sil(struct modesMessage *mm) {
    return mm->accuracy.sil;
}
int modesmessage_get_accuracy_nic_a(struct modesMessage *mm) {
    return mm->accuracy.nic_a;
}
int modesmessage_get_accuracy_nic_b(struct modesMessage *mm) {
    return mm->accuracy.nic_b;
}
int modesmessage_get_accuracy_nic_c(struct modesMessage *mm) {
    return mm->accuracy.nic_c;
}
int modesmessage_get_accuracy_nic_baro(struct modesMessage *mm) {
    return mm->accuracy.nic_baro;
}
int modesmessage_get_accuracy_nac_p(struct modesMessage *mm) {
    return mm->accuracy.nac_p;
}
int modesmessage_get_accuracy_nac_v(struct modesMessage *mm) {
    return mm->accuracy.nac_v;
}
int modesmessage_get_accuracy_gva(struct modesMessage *mm) {
    return mm->accuracy.gva;
}
int modesmessage_get_accuracy_sda(struct modesMessage *mm) {
    return mm->accuracy.sda;
}


int modesmessage_is_nav_heading_valid(struct modesMessage *mm) {
    return mm->nav.heading_valid;
}
int modesmessage_is_nav_fms_altitude_valid(struct modesMessage *mm) {
    return mm->nav.fms_altitude_valid;
}
int modesmessage_is_nav_mcp_altitude_valid(struct modesMessage *mm) {
    return mm->nav.mcp_altitude_valid;
}
int modesmessage_is_nav_qnh_valid(struct modesMessage *mm) {
    return mm->nav.qnh_valid;
}
int modesmessage_is_nav_modes_valid(struct modesMessage *mm) {
    return mm->nav.modes_valid;
}

int modesmessage_is_opstatus_valid(struct modesMessage *mm) {
    return mm->opstatus.valid;
}
int modesmessage_get_opstatus_version(struct modesMessage *mm) {
    return mm->opstatus.version;
}
int modesmessage_is_opstatus_om_acas_ra(struct modesMessage *mm) {
    return mm->opstatus.om_acas_ra;
}
int modesmessage_is_opstatus_om_ident(struct modesMessage *mm) {
    return mm->opstatus.om_ident;
}
int modesmessage_is_opstatus_om_atc(struct modesMessage *mm) {
    return mm->opstatus.om_atc;
}
int modesmessage_is_opstatus_om_saf(struct modesMessage *mm) {
    return mm->opstatus.om_saf;
}
int modesmessage_is_opstatus_cc_acas(struct modesMessage *mm) {
    return mm->opstatus.cc_acas;
}
int modesmessage_is_opstatus_cc_cdti(struct modesMessage *mm) {
    return mm->opstatus.cc_cdti;
}
int modesmessage_is_opstatus_cc_1090_in(struct modesMessage *mm) {
    return mm->opstatus.cc_1090_in;
}
int modesmessage_is_opstatus_cc_arv(struct modesMessage *mm) {
    return mm->opstatus.cc_arv;
}
int modesmessage_is_opstatus_cc_ts(struct modesMessage *mm) {
    return mm->opstatus.cc_ts;
}
int modesmessage_get_opstatus_cc_tc(struct modesMessage *mm) {
    return mm->opstatus.cc_tc;
}
int modesmessage_is_opstatus_cc_uat_in(struct modesMessage *mm) {
    return mm->opstatus.cc_uat_in;
}
int modesmessage_is_opstatus_cc_poa(struct modesMessage *mm) {
    return mm->opstatus.cc_poa;
}
int modesmessage_is_opstatus_cc_b2_low(struct modesMessage *mm) {
    return mm->opstatus.cc_b2_low;
}
int modesmessage_is_opstatus_cc_lw_valid(struct modesMessage *mm) {
    return mm->opstatus.cc_lw_valid;
}
*/
import "C"

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"math"
	"strings"
	"time"
	"unsafe"
)

type (
	// Aircraft simply wraps a readsb aircraft pointer so we can pass it around
	Aircraft struct {
		a *C.struct_aircraft
	}

	// Decoder contains a pointer to the _Modes structure that contains
	// our aircraft state. This state is required for location decoding.
	Decoder struct {
		modes *C.struct__Modes
	}

	// ModesMessage simply wraps a modesMessage pointer so we can pass it around
	// and expose functions to other packages
	ModesMessage struct {
		msg *C.struct_modesMessage
	}
)

var (
	// ErrNoData is returned when the fields data was not available
	ErrNoData = errors.New("no data for field")
)

const (
	ModeACMsgBytes = int(C.MODEAC_MSG_BYTES)

	ModeSShortMsgBytes = int(C.MODES_SHORT_MSG_BYTES)
	ModeSShortMsgBits  = int(C.MODES_SHORT_MSG_BITS)

	ModeSLongMsgBytes = int(C.MODES_LONG_MSG_BYTES)
	ModeSLongMsgBits  = int(C.MODES_LONG_MSG_BITS)

	// Set on addresses to indicate they are not ICAO addresses
	ModeSNonIcaoAddress = int(C.MODES_NON_ICAO_ADDRESS)

	// AsciiIntZero - '0' in ASCII
	AsciiIntZero = 0x30

	ModesReadsbVariant = string(C.MODES_READSB_VARIANT)
)

// IcaoFilterInit calls the readsb function icaoFilterInit which
// initializes an internal filter data structure
func IcaoFilterInit() {
	C.icaoFilterInit()
}
// IcaoFilterExpire should be called periodically so aircraft which
// are out of range (not seen for some TTL) are removed from our filter
func IcaoFilterExpire() {
	C.icaoFilterExpire()
}
// ModeACInit calls the readsb function modeACInit which initializes
// internal conversion tables for modeAC calculations
func ModeACInit() {
	C.modeACInit()
}
// ModesChecksumInit calls the readsb function modesChecksumInit which
// precomputes data about CRC errors
func ModesChecksumInit(numbits int) {
	C.modesChecksumInit(C.int(numbits))
}
// DfToString returns the description of this df value.
func DfToString(df uint) string {
	return C.GoString(C.df_to_string(C.uint(df)))
}

// TrackPeriodicUpdate - Update aircraft state with message info, and update
// message information with supplemental info
func TrackUpdateFromMessage(d *Decoder, mm *ModesMessage) *Aircraft {
	ac := C.trackUpdateFromMessage(d.modes, mm.msg)
	if ac == nil {
		return nil
	}
	return &Aircraft{a: ac}
}

// TrackPeriodicUpdate - Call periodically to remove aircraft who haven't been seen for some TTL
func TrackPeriodicUpdate(d *Decoder) {
	C.trackPeriodicUpdate(d.modes)
}

// GetIcaoHex returns the ICAO as a hex string in upper case
func (m *ModesMessage) GetIcaoHex() string {
	icao := [3]byte{}
	icao[0] = byte(m.msg.addr >> 16)
	icao[1] = byte(m.msg.addr >> 8)
	icao[2] = byte(m.msg.addr >> 0)
	return strings.ToUpper(hex.EncodeToString(icao[:]))
}

// GetSquawk will return the squawk from this message, or ErrNoData if unknown
func (m *ModesMessage) GetSquawk() (string, error) {
	if C.modesmessage_is_squawk_valid(m.msg) != 1 {
		return "", ErrNoData
	}
	return fmt.Sprintf("%04x", uint(m.msg.squawk)), nil
}
// GetCallsign will return the callsign from this message, or ErrNoData if unknown
func (m *ModesMessage) GetCallsign() (string, error) {
	if C.modesmessage_is_callsign_valid(m.msg) != 1 {
		return "", ErrNoData
	}
	return C.GoString((*C.char)(unsafe.Pointer(&m.msg.callsign[0]))), nil
}
// GetAltitudeBaro will return the barometric altitude from this message, or ErrNoData if unknown
func (m *ModesMessage) GetAltitudeBaro() (int64, error) {
	if C.modesmessage_is_altitude_baro_valid(m.msg) != 1 {
		return 0, ErrNoData
	}
	// todo: wtf units are we returning - convert?
	return int64(m.msg.altitude_baro), nil
}
// GetAltitudeGeom will return the geometric altitude from this message, or ErrNoData if unknown
func (m *ModesMessage) GetAltitudeGeom() (int64, error) {
	if C.modesmessage_is_altitude_geom_valid(m.msg) != 1 {
		return 0, ErrNoData
	}

	// todo: wtf units are we returning - convert?
	return int64(m.msg.altitude_geom), nil
}
// GetRateBaro will return the barometric vertical rate from this message, or ErrNoData if unknown
func (m *ModesMessage) GetRateBaro() (int, error) {
	if C.modesmessage_is_baro_rate_valid(m.msg) != 1 {
		return 0, ErrNoData
	}

	// todo: wtf units are we returning - convert?
	return int(m.msg.baro_rate), nil
}
// GetRateGeom will return the geometric vertical rate from this message, or ErrNoData if unknown
func (m *ModesMessage) GetRateGeom() (int, error) {
	if C.modesmessage_is_geom_rate_valid(m.msg) != 1 {
		return 0, ErrNoData
	}

	// todo: wtf units are we returning - convert?
	return int(m.msg.geom_rate), nil
}

// GetGroundSpeed returns the ground speed in knots, or ErrNoData
// if the data is not set.
func (m *ModesMessage) GetGroundSpeed() (float64, error) {
	if C.modesmessage_is_gs_valid(m.msg) != 1 {
		return 0, ErrNoData
	}

	return float64(m.msg.gs.selected), nil
}
// GetDecodeLocation will return the position from this message, or ErrNoData if unknown.
// This field is only set if the message has been processed by TrackUpdateFromMessage as
// to successfully decode a location you need two consecutive odd + even messages.
func (m *ModesMessage) GetDecodeLocation() (float64, float64, error) {
	if C.modesmessage_is_cpr_valid(m.msg) != 1 ||
		C.modesmessage_is_cpr_decoded(m.msg) != 1 {
		return 0, 0, ErrNoData
	}
	lat := float64(m.msg.decoded_lat)
	lon := float64(m.msg.decoded_lon)
	return lat, lon, nil
}
// IsOnGround will return whether the aircraft is on ground, or ErrNoData if
// this is unknown or otherwise uncertain.
func (m *ModesMessage) IsOnGround() (bool, error) {
	if m.msg.airground == C.AIRCRAFT_META__AIR_GROUND__AG_INVALID ||
		m.msg.airground == C.AIRCRAFT_META__AIR_GROUND__AG_UNCERTAIN {
		return false, ErrNoData
	}
	return m.msg.airground == C.AIRCRAFT_META__AIR_GROUND__AG_GROUND, nil
}
// GetHeading returns the heading from the message.
// this field is only set if the mesage has been processed by TrackUpdateFromMEssage
func (m *ModesMessage) GetHeading() (float64, error) {
	if C.modesmessage_is_heading_valid(m.msg) != 1 {
		return 0.0, ErrNoData
	}
	return float64(m.msg.heading), nil
}

// ParseMessage attempts to decode and process any messages it can find in b.
func ParseMessage(d *Decoder, b []byte) ([]*ModesMessage, int, error) {
	var ret []*ModesMessage

	n := len(b)
	som := 0
	eod := n - 1
	var eom int
	for som < eod {
		for som < eod && b[som] != 0x1a {
			som++
		}
		if b[som] != 0x1a {
			break
		}
		p := som + 1
		if p >= eod {
			break
		}
		switch b[p] {
		case AsciiIntZero + 1:
			eom = p + ModeACMsgBytes + 8
		case AsciiIntZero + 2:
			eom = p + ModeSShortMsgBytes + 8
		case AsciiIntZero + 3:
			eom = p + ModeSLongMsgBytes + 8
		case AsciiIntZero + 4:
			eom = p + ModeSLongMsgBytes + 8
		case AsciiIntZero + 5:
			eom = p + ModeSLongMsgBytes + 8
		default:
			som = som + 1
			continue
		}
		for p = som + 1; p < eod && p < eom; p++ {
			if 0x1a == b[p] {
				p++
				eom++
			}
		}

		if eom > eod {
			// incomplete message in buffer, retry later
			break
		}

		mm, err := DecodeBinMessage(d, b[:], som+1, true)
		som = eom
		// this ignores errors from messages, some are just CRC decode
		// errors and so on
		if err == nil {
			ret = append(ret, &ModesMessage{msg: mm})
		}
	}
	return ret, som, nil
}
// DecodeBinMessage attempts to decode a single message, whose starting position
// in m is indicated by p. If withModeAC is true, mode AC messages will be decoded also
func DecodeBinMessage(decoder *Decoder, m []byte, p int, withModeAC bool) (*C.struct_modesMessage, error) {
	var msgLen = 0
	var ch byte
	var j int
	var msg [ModeSLongMsgBytes + 7]byte
	mm := C.struct_modesMessage{}
	ch = m[p]
	p++

	if ch == AsciiIntZero+1 && withModeAC {
		msgLen = ModeACMsgBytes
	} else if ch == AsciiIntZero+2 {
		msgLen = ModeSShortMsgBytes
	} else if ch == AsciiIntZero+3 {
		msgLen = ModeSLongMsgBytes
	} else if ch == AsciiIntZero+5 {
		// special case for radarscape position messages
		//var lat, lon, alt float64
		for j = 0; j < 21; j++ {
			ch = m[p]
			msg[j] = ch
			p++
			if ch == 0x1a {
				p++
			}
		}
		// parse lat
		// parse lon
		// parse alt
	} else {
		return nil, nil
	}

	if msgLen > 0 {
		mm.remote = 1
		mm.timestampMsg = 0
		var t uint64
		for j = 0; j < 6; j++ {
			ch = m[p]
			p++
			t = t<<8 | uint64(ch)
			if 0x1a == ch {
				p++
			}
		}
		mm.timestampMsg = C.ulong(t)
		mm.sysTimestampMsg = C.ulong(time.Now().Unix())

		// grab the signal level
		ch = m[p]
		p++
		var s float64
		s = float64(ch) / 255.0
		s = s * s
		mm.signalLevel = C.double(s)
		if 0x1a == ch {
			p++
		}

		for j = 0; j < msgLen; j++ {
			ch = m[p]
			msg[j] = ch
			p++
			if 0x1a == ch {
				p++
			}
		}

		if msgLen == ModeACMsgBytes {
			mm := C.struct_modesMessage{}
			// is a void function
			C.decodeModeAMessage((*C.struct_modesMessage)(unsafe.Pointer(&mm)), C.int((C.int(msg[0])<<8)|C.int(msg[1])))
			return &mm, nil
		} else {
			mm := C.struct_modesMessage{}
			ret := int(C.decodeModesMessage(decoder.modes, (*C.struct_modesMessage)(unsafe.Pointer(&mm)), (*C.uchar)(unsafe.Pointer(&msg[0]))))
			if ret < 0 {
				if ret == -1 {
					return nil, errors.New("couldn't validate CRC against known ICAO")
				} else if ret == -2 {
					return nil, errors.New("bad message or unrepairable CRC error")
				} else {
					return nil, errors.New("decode error")
				}
			}
			return &mm, nil
		}
	}

	return nil, nil
}

// NewDecoder returns an initialized Decoder.
func NewDecoder() *Decoder {
	return &Decoder{
		modes: &C.struct__Modes{},
	}
}

// DebugModesMessage writes debug information about the message to w.
func DebugModesMessage(w io.Writer, mm *C.struct_modesMessage) error {

	b := C.GoBytes(unsafe.Pointer(&mm.msg[0]), mm.msgbits/8)
	_, err := fmt.Fprintf(w, "msg: %s\n", hex.EncodeToString(b))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "CRC: %d\n", int(mm.crc))
	if mm.correctedbits > 0 {
		_, err = fmt.Fprintf(w, "No. of bit errors fixed: %d\n", mm.correctedbits)
		if err != nil {
			return err
		}
	}

	if mm.signalLevel > 0 {
		rssi := 10 * math.Log10(float64(mm.signalLevel))
		_, err = fmt.Fprintf(w, "RSSI: %f\n", rssi)
		if err != nil {
			return err
		}
	}

	if mm.timestampMsg != 0 {
		if mm.timestampMsg == C.MAGIC_MLAT_TIMESTAMP {
			_, err = fmt.Fprintf(w, "This is a synthetic MLAT message.\n")
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Fprintf(w, "Time: %.2fus\n", float64(mm.timestampMsg)/12.0)
			if err != nil {
				return err
			}
		}
	}

	switch mm.msgtype {
	case 0:
		_, err = fmt.Fprintf(w, "DF:0 addr:%06X VS:%d CC:%d SL:%d RI:%d AC:%d\n",
			mm.addr, mm.VS, mm.CC, mm.SL, mm.RI, mm.AC)
		if err != nil {
			return err
		}
	case 4:
		_, err = fmt.Fprintf(w, "DF:4 addr:%06X FS:%d DR:%d UM:%d AC:%d\n",
			mm.addr, mm.FS, mm.DR, mm.UM, mm.AC)
		if err != nil {
			return err
		}
	case 5:
		_, err = fmt.Fprintf(w, "DF:5 addr:%06X FS:%d DR:%d UM:%d ID:%d\n",
			mm.addr, mm.FS, mm.DR, mm.UM, mm.ID)
		if err != nil {
			return err
		}
	case 11:
		_, err = fmt.Fprintf(w, "DF:11 AA:%06X IID:%d CA:%d\n",
			mm.AA, mm.IID, mm.CA)
		if err != nil {
			return err
		}
	case 16:
		_, err = fmt.Fprintf(w, "DF:16 addr:%06x VS:%d SL:%d RI:%d AC:%d MV:",
			mm.addr, mm.VS, mm.SL, mm.RI, mm.AC)
		if err != nil {
			return err
		}
		mv := C.GoBytes(unsafe.Pointer(&mm.MV[0]), 7)
		_, err = fmt.Fprintf(w, hex.EncodeToString(mv))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	case 17:
		_, err = fmt.Fprintf(w, "DF:17 AA:%06X CA:%d ME:",
			mm.AA, mm.CA)
		if err != nil {
			return err
		}
		me := C.GoBytes(unsafe.Pointer(&mm.ME[0]), 7)
		_, err = fmt.Fprintf(w, hex.EncodeToString(me))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	case 18:
		_, err = fmt.Fprintf(w, "DF:18 AA:%06X CF:%d ME:",
			mm.AA, mm.CF)
		if err != nil {
			return err
		}
		me := C.GoBytes(unsafe.Pointer(&mm.ME[0]), 7)
		_, err = fmt.Fprintf(w, hex.EncodeToString(me))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	case 20:
		_, err = fmt.Fprintf(w, "DF:20 addr:%06X FS:%d DR:%d UM:%d AC:%d MB:",
			mm.addr, mm.FS, mm.DR, mm.UM, mm.AC)
		if err != nil {
			return err
		}
		mb := C.GoBytes(unsafe.Pointer(&mm.MB[0]), 7)
		_, err = fmt.Fprintf(w, hex.EncodeToString(mb))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	case 21:
		_, err = fmt.Fprintf(w, "DF:21 addr:%06x FS:%d DR:%d UM:%d ID:%d MB:",
			mm.addr, mm.FS, mm.DR, mm.UM, mm.ID)
		if err != nil {
			return err
		}
		mb := C.GoBytes(unsafe.Pointer(&mm.MB[0]), 7)
		_, err = fmt.Fprintf(w, hex.EncodeToString(mb))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	case 24:
	case 25:
	case 26:
	case 27:
	case 28:
	case 29:
	case 30:
	case 31:
		_, err = fmt.Fprintf(w, "DF:24 addr:%06x KE:%d ND:%d MD:",
			mm.addr, mm.KE, mm.ND)
		if err != nil {
			return err
		}
		md := C.GoBytes(unsafe.Pointer(&mm.MD[0]), 10)
		_, err = fmt.Fprintf(w, hex.EncodeToString(md))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "%s", C.GoString(C.df_to_string(C.uint(mm.msgtype))))
	if err != nil {
		return err
	}
	if mm.msgtype == 17 || mm.msgtype == 18 {
		if C.esTypeHasSubtype(mm.metype) == 1 {
			_, err = fmt.Fprintf(w, " %s (%d/%d)",
				C.GoString(C.esTypeName(mm.metype, mm.mesub)),
				mm.metype,
				mm.mesub)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Fprintf(w, " %s (%d)",
				C.GoString(C.esTypeName(mm.metype, mm.mesub)),
				int(mm.metype))
			if err != nil {
				return err
			}
		}
	}

	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		return err
	}

	if mm.msgtype == 20 || mm.msgtype == 21 {
		_, err = fmt.Fprintf(w, "  Comm-B format: %s\n", C.GoString(C.commb_format_to_string(mm.commb_format)))
		if err != nil {
			return err
		}
	}

	if (int(mm.addr) & ModeSNonIcaoAddress) != 0 {
		_, err = fmt.Fprintf(w, "  Other Address: %06X (%s)\n", mm.addr&0xFFFFFF, C.GoString(C.addrtype_to_string(mm.addrtype)))
		if err != nil {
			return err
		}
	} else {
		_, err = fmt.Fprintf(w, "  ICAO Address:  %06X (%s)\n", mm.addr, C.GoString(C.addrtype_to_string(mm.addrtype)))
		if err != nil {
			return err
		}
	}

	if mm.airground != C.AIRCRAFT_META__AIR_GROUND__AG_INVALID {
		_, err = fmt.Fprintf(w, "  Air/Ground:    %s\n",
			C.GoString(C.airground_to_string(mm.airground)))
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_altitude_baro_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Baro altitude: %d %s\n",
			mm.altitude_baro,
			C.GoString(C.altitude_unit_to_string(mm.altitude_baro_unit)))
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_altitude_geom_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Geom altitude: %d %s\n",
			mm.altitude_geom,
			C.GoString(C.altitude_unit_to_string(mm.altitude_geom_unit)))
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_geom_delta_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Geom - baro:   %d ft\n", int(mm.geom_delta))
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_heading_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  %-13s  %.1f\n", C.GoString(C.heading_type_to_string(mm.heading_type)), mm.heading)
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_track_rate_valid(mm) == 1 {
		var direction string
		if mm.track_rate < 0 {
			direction = "left"
		} else if mm.track_rate > 0 {
			direction = "right"
		}
		_, err = fmt.Fprintf(w, "  Track rate:    %.2f deg/sec %s\n", float64(mm.track_rate), direction)
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_roll_valid(mm) == 1 {
		var direction string
		if mm.roll < -0.05 {
			direction = "left"
		} else if mm.roll > 0.05 {
			direction = "right"
		}
		_, err = fmt.Fprintf(w, "  Roll:          %.1f degrees %s\n", float64(mm.roll), direction)
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_gs_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Groundspeed:   %.1f kt", float64(mm.gs.selected))
		if err != nil {
			return err
		}
		if mm.gs.v0 != mm.gs.selected {
			_, err = fmt.Fprintf(w, " (v0: %.1f kt)", float64(mm.gs.v0))
			if err != nil {
				return err
			}
		}
		if mm.gs.v2 != mm.gs.selected {
			_, err = fmt.Fprintf(w, " (v2: %.1f kt)", float64(mm.gs.v2))
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_ias_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  IAS:           %d kt\n", uint(mm.ias))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_tas_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  TAS:           %d kt\n", uint(mm.tas))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_mach_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Mach number:   %.3f\n", float64(mm.mach))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_baro_rate_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Baro rate:     %d ft/min\n", uint(mm.baro_rate))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_geom_rate_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Geom rate:     %d ft/min\n", uint(mm.geom_rate))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_squawk_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Squawk:        %04x\n", uint(mm.squawk))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_callsign_valid(mm) == 1 {
		ident := C.GoString((*C.char)(unsafe.Pointer(&mm.callsign[0])))
		_, err = fmt.Fprintf(w, "  Ident:         %s\n", ident)
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_category_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Category:      %02X\n", uint(mm.category))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_cpr_valid(mm) == 1 {
		oddOrEven := "even"
		if C.modesmessage_is_cpr_odd(mm) == 1 {
			oddOrEven = "even"
		}
		_, err = fmt.Fprintf(w, "  CPR type:      %s\n", C.GoString(C.cpr_type_to_string(mm.cpr_type)))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "  CPR odd flag:  %s\n", oddOrEven)
		if err != nil {
			return err
		}
		//
		if int(C.modesmessage_is_cpr_decoded(mm)) == 1 {
			cprDecoding := "global"
			if int(C.modesmessage_is_cpr_relative(mm)) == 1 {
				cprDecoding = "local"
			}

			_, err = fmt.Fprintf(w, "  CPR latitude:  %.5f (%f)\n"+
				"  CPR longitude: %.5f (%f)\n"+
				"  CPR decoding:  %s\n"+
				"  NIC:           %d\n"+
				"  Rc:            %.3f km / %.1f NM\n",
				float64(mm.decoded_lat),
				float64(mm.cpr_lat),
				float64(mm.decoded_lon),
				float64(mm.cpr_lon),
				cprDecoding,
				uint(mm.decoded_nic),
				float64(mm.decoded_rc/1000.0),
				float64(mm.decoded_rc/1852.0))
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Fprintf(w, "  CPR latitude:  (%f)\n"+
				"  CPR longitude: (%f)\n"+
				"  CPR decoding:  none\n",
				float64(mm.cpr_lat),
				float64(mm.cpr_lon))
			if err != nil {
				return err
			}
		}
	}

	if C.modesmessage_is_accuracy_nic_a_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NIC-A:         %d\n", C.modesmessage_get_accuracy_nic_a(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_nic_b_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NIC-B:         %d\n", C.modesmessage_get_accuracy_nic_b(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_nic_c_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NIC-C:         %d\n", C.modesmessage_get_accuracy_nic_c(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_nic_baro_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NIC-baro:      %d\n", C.modesmessage_get_accuracy_nic_baro(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_nac_p_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NACp:          %d\n", C.modesmessage_get_accuracy_nac_p(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_nac_v_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NACv:          %d\n", C.modesmessage_get_accuracy_nac_v(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_gva_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  GVA:           %d\n", C.modesmessage_get_accuracy_gva(mm))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_nic_c_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  NIC-C:         %d\n", C.modesmessage_get_accuracy_nic_c(mm))
		if err != nil {
			return err
		}
	}

	if mm.accuracy.sil_type != C.AIRCRAFT_META__SIL_TYPE__SIL_INVALID {
		var silDescription string
		switch C.modesmessage_get_accuracy_sil(mm) {
		case 1:
			silDescription = "p <= 0.1%"
		case 2:
			silDescription = "p <= 0.001%"
		case 3:
			silDescription = "p <= 0.00001%"
		default:
			silDescription = "p > 0.1%"
		}
		_, err = fmt.Fprintf(w, "  SIL:           %d (%s, %s)\n",
			int(C.modesmessage_get_accuracy_sil(mm)),
			silDescription,
			C.GoString(C.sil_type_to_string(mm.accuracy.sil_type)))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_accuracy_sda_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  SDA:           %d\n", int(C.modesmessage_get_accuracy_sda(mm)))
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_opstatus_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Aircraft Operational Status:\n")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "    Version:            %d\n", C.modesmessage_get_opstatus_version(mm))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "    Capability classes: ")
		if err != nil {
			return err
		}
		if C.modesmessage_is_opstatus_cc_acas(mm) == 1 {
			_, err = fmt.Fprintf(w, "ACAS ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_cdti(mm) == 1 {
			_, err = fmt.Fprintf(w, "CDTI ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_1090_in(mm) == 1 {
			_, err = fmt.Fprintf(w, "1090IN ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_arv(mm) == 1 {
			_, err = fmt.Fprintf(w, "ARV ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_ts(mm) == 1 {
			_, err = fmt.Fprintf(w, "TS ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_get_opstatus_cc_tc(mm) != 0 {
			_, err = fmt.Fprintf(w, "TC=%d ", C.modesmessage_get_opstatus_cc_tc(mm))
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_uat_in(mm) == 1 {
			_, err = fmt.Fprintf(w, "UATIN ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_poa(mm) == 1 {
			_, err = fmt.Fprintf(w, "POA ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_b2_low(mm) == 1 {
			_, err = fmt.Fprintf(w, "B2-LOW ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_cc_lw_valid(mm) == 1 {
			_, err = fmt.Fprintf(w, "L/W=%d  ", mm.opstatus.cc_lw)
			if err != nil {
				return err
			}
		}
		if mm.opstatus.cc_antenna_offset != 0 {
			_, err = fmt.Fprintf(w, "GPS-OFFSET=%d ", mm.opstatus.cc_antenna_offset)
			if err != nil {
				return err
			}
		}

		_, err = fmt.Fprintf(w, "    Operational modes:  ")
		if err != nil {
			return err
		}
		if C.modesmessage_is_opstatus_om_acas_ra(mm) == 1 {
			_, err = fmt.Fprintf(w, "ACASRA ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_om_ident(mm) == 1 {
			_, err = fmt.Fprintf(w, "IDENT ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_om_atc(mm) == 1 {
			_, err = fmt.Fprintf(w, "ATC ")
			if err != nil {
				return err
			}
		}
		if C.modesmessage_is_opstatus_om_saf(mm) == 1 {
			_, err = fmt.Fprintf(w, "SAF ")
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
		if mm.mesub == 1 {
			_, err = fmt.Fprintf(w, "    Track/heading:      %s\n", C.GoString(C.heading_type_to_string(mm.opstatus.tah)))
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(w, "    Heading ref dir:    %s\n", C.GoString(C.heading_type_to_string(mm.opstatus.hrd)))
		if err != nil {
			return err
		}
	}

	if C.modesmessage_is_nav_heading_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Selected heading:        %.1f\n", float64(mm.nav.heading))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_nav_fms_altitude_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  FMS selected altitude:   %d ft\n", uint(mm.nav.fms_altitude))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_nav_mcp_altitude_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  MCP selected altitude:   %d ft\n", uint(mm.nav.mcp_altitude))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_nav_qnh_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  QNH:                     %.1f millibars\n", float64(mm.nav.qnh))
		if err != nil {
			return err
		}
	}
	if mm.nav.altitude_source != C.NAV_ALT_INVALID {
		_, err = fmt.Fprintf(w, "  Target altitude source:  ")
		if err != nil {
			return err
		}
		switch mm.nav.altitude_source {
		case C.NAV_ALT_AIRCRAFT:
			_, err = fmt.Fprintf(w, "aircraft altitude\n")
			if err != nil {
				return err
			}
		case C.NAV_ALT_MCP:
			_, err = fmt.Fprintf(w, "MCP selected altitude\n")
			if err != nil {
				return err
			}
		case C.NAV_ALT_FMS:
			_, err = fmt.Fprintf(w, "FMS selected altitude\n")
			if err != nil {
				return err
			}
		default:
			_, err = fmt.Fprintf(w, "unknown\n")
			if err != nil {
				return err
			}
		}
	}
	if C.modesmessage_is_nav_modes_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Nav modes:               %s\n", C.GoString(C.nav_modes_to_string(mm.nav.modes)))
		if err != nil {
			return err
		}
	}
	if C.modesmessage_is_emergency_valid(mm) == 1 {
		_, err = fmt.Fprintf(w, "  Emergency/priority:      %s\n", C.GoString(C.emergency_to_string(mm.emergency)))
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		return err
	}
	return nil
}
