package main

import "C"
import (
	"encoding/hex"
	"fmt"
	"github.com/afk11/airtrack/pkg/readsb"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		panic("missing message hex")
	}
	df := uint(0)
	fmt.Println(readsb.DfToString(df))
	fmt.Println(readsb.ModesReadsbVariant)
	readsb.IcaoFilterInit()
	readsb.ModeACInit()
	readsb.ModesChecksumInit(1)
	b, err := hex.DecodeString(os.Args[1]) // message
	if err != nil {
		panic(err)
	}

	//b, err := hex.DecodeString("1a33"+//msgType
	//	"0031acd41922" + //timestamp
	//	"18"+//signal strength
	//	"8da9bd9a990d75a7d80464d68ee7") // message
	//b, err := hex.DecodeString("1a3200390d1fae22215d89913cfb39600090955d32fd7f") // message
	//mm := C.struct_modesMessage{}
	//var modes *C.struct__Modes = nil
	//ret := int(C.decodeModesMessage(modes, (*C.struct_modesMessage)(unsafe.Pointer(&mm)), (*C.uchar)(unsafe.Pointer(&b[0]))))

	_, _, err 	= readsb.ParseMessage(b)
	if err != nil {
		panic(err)
	}
}