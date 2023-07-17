package decode

import (
	"fmt"

	"github.com/metarex-media/mrx-tool/klv"
)

type keyLengthDecode struct {
	keyLen, lengthLen int
	lengthFunc        func([]byte) (int, int)
	keyFunc           func([]byte) (string, int)
}

func oneNameKL(namebytes []byte) (string, int) {
	if len(namebytes) != 1 {
		return "", 0
	}

	return fmt.Sprintf("%02x", namebytes[0:1:1]), 1
}

func oneLengthKL(lengthbytes []byte) (int, int) {
	if len(lengthbytes) != 1 {
		return 0, 0
	}

	return int(lengthbytes[0]), 1
}

func twoNameKL(namebytes []byte) (string, int) {
	if len(namebytes) != 2 {
		return "", 0
	}

	return fmt.Sprintf("%04x", namebytes[0:2:2]), 2
}

func twoLengthKL(lengthbytes []byte) (int, int) {
	if len(lengthbytes) != 2 {
		return 0, 0
	}

	length := order.Uint16(lengthbytes[0:2:2])

	return int(length), 2
}

func fullNameKL(namebytes []byte) (string, int) {

	if len(namebytes) != 16 {
		return "", 0
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15]), 16
}

// decodeBuilder generates the options to decode a packet.
// some tags need to be updated. If the value is not known then bool is true for you to skip this packet
func decodeBuilder(key uint8) (keyLengthDecode, bool) {
	var decodeOption keyLengthDecode
	var skip bool
	lenField := (key >> 4)
	keyField := (key & 0b00001111)

	// smpte 336 decode methods
	switch lenField {
	case 0, 1:
		decodeOption.lengthLen = 16
		decodeOption.lengthFunc = klv.BerDecode
	case 4, 5:
		decodeOption.lengthLen = 2
		decodeOption.lengthFunc = twoLengthKL
	default:
		skip = true
	}

	switch lenField%2 + keyField {
	case 0, 1, 2, 0xB:
		decodeOption.keyFunc = fullNameKL
		decodeOption.keyLen = 16
	case 4:
		decodeOption.keyFunc = twoNameKL
		decodeOption.keyLen = 2
	case 3:
		decodeOption.keyFunc = oneNameKL
		decodeOption.keyLen = 1
	case 0xC:
		// 3 is 1 byte
		// 0xB is ASN1
		// 0xC is 4
	default:
		skip = true
	}

	return decodeOption, skip
}

func primerUnpack(input []byte, shorthand map[string]string) {

	count := order.Uint32(input[0:4])
	length := order.Uint32(input[4:8]) // if length isn't 18 explode

	offset := 8
	for i := uint32(0); i < count; i++ {
		//fmt.Printf("%x: %v\n", input[offset:offset+2], fullName(input[offset+2:offset+18]))
		short := fmt.Sprintf("%04x", input[offset:offset+2])
		shorthand[short] = fullName(input[offset+2 : offset+18])
		offset += int(length)
	}

}
