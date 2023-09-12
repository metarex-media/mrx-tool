package klv

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/stream"
	"golang.org/x/sync/errgroup"
)

func BufferWrap(fStream io.Reader, klvStream chan *KLV, size int) error {

	bufferStream := make(chan *stream.Packet, 1*size)

	errs, _ := errgroup.WithContext(context.Background())

	//initiate the klv stream
	errs.Go(func() error {
		return stream.BufferManager(fStream, bufferStream, size)

	})

	//go stream.BufferManager(fStream, size, bufferStream)

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

func klvEncode() {

	// simple way of handling all these bits
	// just smash them until the packet size is reached
}

/*
func KLV encode that goes the other way
// wonder if theres a buffer that will be generated to fit it
*/

func klvDecode(buffer chan *stream.Packet, klvOut chan *KLV) error { //wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	defer close(klvOut)

	// the design of this is to return a simple KLV that can the be handled however we so desire

	partStreamP, streamOpen := <-buffer

	var partStream []byte
	if !streamOpen { // stop pointer errors
		return fmt.Errorf("Empty data stream")
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

		section.Key = keyBytes
		// check there's enough file to get a key and length
		// if not extend the buffer
		/*if len(partStream) < position+16 {
			//	fmt.Println("trigger", position+16, len(partStream))
			section.Key = streamContents.partStream[position:]
			//	fmt.Println(partStream[position:])
			runoff := position + 16 - len(streamContents.partStream)
			partStreamP, streamOpen = <-streamContents.buffer
			if !streamOpen {

				break
			}
			streamContents.partStream = partStreamP.Packet

			if !streamOpen {
				//	errChan <- fmt.Errorf("Data stream unexpectedly closed")
				return fmt.Errorf("Data stream unexpectedly closed")
			}
			//	fmt.Println(runoff, partStream[:runoff])
			section.Key = append(section.Key, partStream[:runoff]...)
			position = runoff

		} else {
			section.Key = partStream[position : position+16 : position+16]
			position += 16
		}*/

		// get the BERDecode length
		/*if position >= len(streamContents.partStream) {
			partStreamP, streamOpen = <-streamContents.buffer
			if !streamOpen {
				return fmt.Errorf("unexpected end of data stream")
			}
			streamContents.partStream = partStreamP.Packet
			position = 0
		}*/

		berDecodeLength := 1 + (berLength(streamContents.partStream[position]))
		lengthBytes, err := streamContents.bridger(&position, berDecodeLength)
		if err != nil {
			return err
		}

		section.Length = lengthBytes
		/*
			if len(partStream) < position+berDecodeLength {
				section.Length = partStream[position:]
				//	fmt.Println(partStream[position:])
				runoff := position + berDecodeLength - len(partStream)
				partStreamP, streamOpen = <-buffer
				if !streamOpen {
					break
				}
				partStream = partStreamP.Packet
				if !streamOpen {
					//	errChan <- fmt.Errorf("Data stream unexpectedly closed")
					return fmt.Errorf("Data stream unexpectedly closed")
				}
				//	fmt.Println(runoff, partStream[:runoff])
				section.Length = append(section.Key, partStream[:runoff]...)
				position = runoff

			} else {
				section.Length = partStream[position : position+berDecodeLength : position+berDecodeLength]
				position += berDecodeLength
			}*/

		partLength, _ := BerDecode(section.Length)

		//	section.Length = section.Length[:BERlength:BERlength]

		section.LengthValue = partLength

		valueBytes, err := streamContents.bridger(&position, partLength)
		if err != nil {
			return err
		}

		section.Value = valueBytes

		// go through the stream until all the data has been hooverd up
		/*
			remain := partLength
			endPosition := position + partLength

			if endPosition > len(streamContents.partStream) {
				endPosition = len(streamContents.partStream)
			}
			//	fmt.Println("STOP2")
			//position += +BERlength
			for remain > 0 {
				//fmt.Println(remain, position, endPosition, "BER LENGTH", section.key, total)

				section.Value = append(section.Value, streamContents.partStream[position:endPosition:endPosition]...)

				remain -= (endPosition - position)
				if endPosition == len(streamContents.partStream) {
					position = 0
					endPosition = remain
					partStreamP, streamOpen = <-streamContents.buffer
					if !streamOpen {
						if remain != 0 {
							return fmt.Errorf("Buffer stream unexpectantly closed, was expecting at least %v bytes", remain)
						}

						break
					}
					streamContents.partStream = partStreamP.Packet
					if endPosition > len(streamContents.partStream) {
						endPosition = len(streamContents.partStream)
					}
				} else {
					position = endPosition
				}
			}
		*/
		//	fmt.Println(section)
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

func (s *streamer) bridger(positionPoint *int, bridgeSize int) ([]byte, error) {
	position := *positionPoint
	remain := bridgeSize
	bridged := []byte{}

	endPosition := position + bridgeSize

	if endPosition > len(s.partStream) {
		endPosition = len(s.partStream)
	}
	//	fmt.Println("STOP2")
	//position += +BERlength
	for remain > 0 {
		//fmt.Println(remain, position, endPosition, "BER LENGTH", section.key, total)

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

			//else keep hoovering up the stream
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

/*
// bridgeBytes searches across the buffer for when the data runs across a data point.
func (s *streamer) bridgeBytes(positionPoint *int, bridgeSize int) ([]byte, error) {
	var bridged []byte

	position := *positionPoint
	if len(s.partStream) < position+bridgeSize {
		bridged = s.partStream[position:]
		//	fmt.Println(partStream[position:])
		runoff := position + bridgeSize - len(s.partStream)
		partStreamP, streamOpen := <-s.buffer
		fmt.Println(position, len(s.partStream))
		if !streamOpen {
			//	errChan <- fmt.Errorf("Data stream unexpectedly closed")
			return bridged, fmt.Errorf("Data stream unexpectedly closed")
		}
		s.partStream = partStreamP.Packet
		//	fmt.Println(runoff, partStream[:runoff])
		bridged = append(bridged, s.partStream[:runoff]...)
		position = runoff

	} else {
		bridged = s.partStream[position : position+bridgeSize : position+bridgeSize]
		position += bridgeSize
	}

	*positionPoint = position
	return bridged, nil
}*/

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

// BerDecode decodes BERenocded lengths up to 9 bytes long
// including the indentifier byte.
func BerDecode(num []byte) (length int, encodeLength int) {

	if len(num) == 0 {
		return 0, 0
	}
	// mxf doesn;t exceed a length of 9
	// which is 1 giving the lenght and 8 bytes
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
		//lengthproxy := int(length)
		postion := 7

		if int(length) > len(num)-1 {
			length = uint8(len(num) - 1)
		}

		for lengthproxy := int(length); lengthproxy > 0; lengthproxy-- {
			complete[postion] = num[lengthproxy]
			postion--
			//lengthproxy--
		}

		// 8 is the identifier
		return int(order.Uint64(complete)), int(length + 1)
	}

}
