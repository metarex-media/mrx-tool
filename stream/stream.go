package stream

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

var order = binary.BigEndian

type Packet struct {
	Packet   []byte
	Position int
}

func BufferManager(stream io.Reader, bufferStream chan *Packet, size int) error {
	bufferDiv := size

	// TODO implement a set bufferDivision
	sizer := 104857600 / bufferDiv // 100

	bufReader := bufio.NewReaderSize(stream, sizer)

	//count labels
	count := 0
	for {
		// TODO multiplexer requred for true streams

		bufferPacket := make([]byte, sizer)
		bufFill, err := bufReader.Read(bufferPacket)

		// insert a method for reading here which truncates
		// the packet if not all the bytes are written

		if err != nil {
			//fmt.Println(err)
			//	quit <- false
			close(bufferStream)
			if err == io.EOF {
				return nil
			} else {
				return fmt.Errorf("error reading and buffering data %v", err)
			}
		}

		// write only the bytes needed, if the buffer is half filled then send that
		bufferStream <- &Packet{Position: count, Packet: bufferPacket[:bufFill]}

		count++
	}

}
