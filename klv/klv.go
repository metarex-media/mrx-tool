package klv

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/stream"
	"golang.org/x/sync/errgroup"
)

// StartKLVStream breaks the reader into a stream of the MRX klv values.
func StartKLVStream(fStream io.Reader, klvStream chan *KLV, size int) error {

	bufferStream := make(chan *stream.Packet, 1*size)

	errs, _ := errgroup.WithContext(context.Background())

	// initiate the stream of packets
	errs.Go(func() error {
		return stream.BufferManager(fStream, bufferStream, size)

	})

	// decode the packets to their klv values
	errs.Go(func() error {
		return klvDecode(bufferStream, klvStream)

	})

	return errs.Wait()

}

type KLV struct {
	Key    []byte
	Length []byte
	Value  []byte

	// Length Value gives the value of the length to not redecode BER
	LengthValue int
}

/*
func KLV encode that goes the other way
// wonder if theres a buffer that will be generated to fit it
*/

func klvDecode(buffer chan *stream.Packet, klvOut chan *KLV) error { // wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	defer close(klvOut)

	// the design of this is to return a simple KLV that can the be handled however we so desire

	partStreamP, streamOpen := <-buffer

	var partStream []byte
	if !streamOpen { // stop pointer errors
		return fmt.Errorf("empty data stream")
	}

	partStream = partStreamP.Packet
	position := 0
	// loop through all the buffers

	streamContents := streamer{partStream: partStream, buffer: buffer, streamOpen: streamOpen}

	for streamContents.streamOpen {

		//	fmt.Println(position)
		var section = KLV{Value: []byte{}}

		// get the key value
		keyBytes, err := streamContents.bridger(&position, 16)
		if err != nil {
			return err
		}

		// set the key
		section.Key = keyBytes

		berDecodeLength := 1 + (berLength(streamContents.partStream[position]))
		lengthBytes, err := streamContents.bridger(&position, berDecodeLength)
		if err != nil {
			return err
		}

		// Set the length
		section.Length = lengthBytes

		// find the value bytes
		partLength, _ := BerDecode(section.Length)
		section.LengthValue = partLength

		valueBytes, err := streamContents.bridger(&position, partLength)
		if err != nil {
			return err
		}

		section.Value = valueBytes

		// return klv Section
		klvOut <- &section
	}

	return nil
}

type streamer struct {
	partStream []byte
	buffer     chan *stream.Packet
	streamOpen bool
}

// bridger bridges the bytes between two packets
func (s *streamer) bridger(positionPoint *int, bridgeSize int) ([]byte, error) {
	position := *positionPoint
	remain := bridgeSize
	bridged := []byte{}

	endPosition := position + bridgeSize

	if endPosition > len(s.partStream) {
		endPosition = len(s.partStream)
	}

	for remain > 0 {
		// fmt.Println(remain, position, endPosition, "BER LENGTH", section.key, total)

		bridged = append(bridged, s.partStream[position:endPosition:endPosition]...)

		remain -= (endPosition - position)
		if endPosition == len(s.partStream) {
			position = 0
			endPosition = remain
			partStreamP, streamOpen := <-s.buffer
			s.streamOpen = streamOpen
			if !streamOpen {
				if remain != 0 {
					return bridged, fmt.Errorf("Buffer stream unexpectantly closed, was expecting at least %v more bytes", remain)
				}

				return bridged, nil
			}

			// else keep hoovering up the stream
			s.partStream = partStreamP.Packet
			if endPosition > len(s.partStream) {
				endPosition = len(s.partStream)
			}

		} else {
			position = endPosition
		}
	}

	*positionPoint = position
	return bridged, nil
}

// TotalLength returns the total length of a klv packet
func (k *KLV) TotalLength() int {

	return len(k.Key) + len(k.Length) + len(k.Value)
}

var order = binary.BigEndian

func berLength(length byte) int {
	if length < 0x7f {
		return 0
	}

	// take the 4 lsbf for the length
	return int(0x0f & length)
}

// BerDecode decodes BERencoded lengths up to 9 bytes long
// including the indentifier byte.
func BerDecode(num []byte) (length int, encodeLength int) {

	if len(num) == 0 {
		return 0, 0
	}
	// mxf doesn;t exceed a length of 9
	// which is 1 giving the length and 8 bytes
	start := num[0]
	if start < 0x7f {
		return int(start), 1
	} else {

		// take the 4 lsbf for the length
		length := 0x0f & start

		if length > 8 {
			return 0, 0
		}

		complete := make([]byte, 8)
		// lengthproxy := int(length)
		postion := 7

		if int(length) > len(num)-1 {
			length = uint8(len(num) - 1)
		}

		for lengthproxy := int(length); lengthproxy > 0; lengthproxy-- {
			complete[postion] = num[lengthproxy]
			postion--
			// lengthproxy--
		}

		// 8 is the identifier
		return int(order.Uint64(complete)), int(length + 1)
	}

}
