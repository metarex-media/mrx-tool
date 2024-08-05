package stream

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGoodStream(t *testing.T) {

	// os.Create("./testdata/testfile.txt")

	setErr := fileMake([]string{"./testdata/testfile.txt"})
	Convey("Setting test files to read", t, func() {
		Convey("setting up  test file with random bytes", func() {
			Convey("no error is generated and the test file is set up", func() {
				So(setErr, ShouldBeNil)
			})
		})
	})

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

func fileMake(files []string) error {

	for _, fn := range files {
		dir, _ := filepath.Abs(fn)
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}

		fb := make([]byte, 250000)
		rand.Read(fb)
		f, err := os.Create(fn)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(fb)
		if err != nil {
			return err
		}
	}

	return nil
}
