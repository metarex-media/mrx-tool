// Package stream handles data streaming functions
package stream

import (
	"bufio"
	"fmt"
	"io"
)

type Packet struct {
	Packet   []byte
	Position int
}

// BufferManager splits a reader into packets, each packet is the size of
// 10mb / size
func BufferManager(stream io.Reader, bufferStream chan *Packet, size int) error {
	bufferDiv := size

	// TODO implement a set bufferDivision
	sizer := 104857600 / bufferDiv // 100

	bufReader := bufio.NewReaderSize(stream, sizer)

	// count labels
	count := 0
	for {
		// @TODO multiplexer requred for true streams

		bufferPacket := make([]byte, sizer)
		bufFill, err := bufReader.Read(bufferPacket)

		if err != nil {
			// fmt.Println(err)
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
