package stream

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGoodStream(t *testing.T) {

	// os.Create("./testdata/testfile.txt")

	fileMake([]string{"./testdata/testfile.txt"})

	f, _ := os.Open("./testdata/testfile.txt")

	gen := BufferManager(f, make(chan *Packet, 100), 10000)

	Convey("Checking that a file stream can be read", t, func() {
		Convey("using a generated file with no expected errors", func() {
			Convey("no error is generated and all the file is extracted", func() {
				So(gen, ShouldBeNil)
			})
		})
	})

}

type breaker struct {
}

func (b breaker) Read(in []byte) (int, error) {
	return 0, os.ErrClosed
}

func TestBadStream(t *testing.T) {

	// os.Create("./testdata/testfile.txt")

	gen := BufferManager(breaker{}, make(chan *Packet, 100), 10000)

	Convey("Checking that a sudden stop of the stream is handled", t, func() {
		Convey("running the stream which always returns an error", func() {
			Convey("the error is caught and the stream is stopped", func() {
				So(gen, ShouldResemble, fmt.Errorf("error reading and buffering data file already closed"))
			})
		})
	})

}

func fileMake(files []string) {
	for _, fn := range files {
		fb := make([]byte, 250000)
		rand.Read(fb)
		f, _ := os.Create(fn)
		defer f.Close()
		f.Write(fb)
	}
}
