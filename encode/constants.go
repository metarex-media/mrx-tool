package encode

// EssenceKey is the id for the metadata keys
type EssenceKey uint64

const (
	// TextFrame is for clocked text metadata
	TextFrame EssenceKey = iota + 1
	// TextClip is for embedded text metadata
	TextClip
	// TextFrame is for clocked binary metadata
	BinaryFrame
	// TextClip is for embedded binary metadata
	BinaryClip
)

/*
var genericKeyText = [16]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0000, 0, 0, 0}
var genericKeyBinary = [16]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0001, 0, 0, 0}
var binarycFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x7f, 0x00, 0x00}
var textcFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x7f, 0x01, 0x01}*/

func getKeyBytes(key EssenceKey) []byte {

	var genericKeyText = [16]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0000, 0, 0, 0}
	var genericKeyBinary = [16]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0001, 0, 0, 0}
	var binarycFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x7f, 0x00, 0x00}
	var textcFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x7f, 0x01, 0x01}

	switch key {
	case TextFrame:
		return textcFrameKey[:]
	case TextClip:
		return genericKeyText[:]
	case BinaryFrame:
		return binarycFrameKey[:]
	case BinaryClip:
		return genericKeyBinary[:]

	default:
		return []byte{}

	}

}

var isxdContainer = [16]byte{06, 0x0E, 0x2B, 0x34, 04, 01, 01, 05, 0x0E, 0x09, 06, 07, 01, 01, 01, 03}
var genericContainer = [16]byte{06, 0x0e, 0x2b, 0x34, 04, 01, 01, 03, 0x0d, 01, 03, 01, 02, 0x7f, 01, 00}

func getContainerKey(key EssenceKey) []byte {

	switch key {
	case TextFrame:
		return isxdContainer[:]
	case TextClip:
		// @TODO update the key
		return genericContainer[:]
	case BinaryFrame, BinaryClip:
		return genericContainer[:]

	default:
		return []byte{}

	}
}
