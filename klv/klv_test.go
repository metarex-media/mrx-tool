package klv

import (
	"fmt"
	"io"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileRead(t *testing.T) {

	// empty, _ = os.Create("result/helpme.yaml")
	// stream, _ := os.Open("../mrx-starter/examples/newtests/Disney_Test_Patterns_ISXD.mxf")
	streamer, _ := os.Open("../testdata/rexy_sunbathe_mrx.mxf")
	noError := BufferWrap(streamer, make(chan *KLV, 9000), 10)

	Convey("Checking that a klv stream is read", t, func() {
		Convey("running the stream to fill up the klv system", func() {
			Convey("no error is produced and every klv is produced", func() {
				So(noError, ShouldBeNil)
			})
		})
	})

	streamer2, _ := os.Open("../testdata/rexy_sunbathe_mrx.mxf")
	noError = BufferWrap(streamer2, make(chan *KLV, 9000), 1000000)

	Convey("Checking that a klv stream is read, from a data stream of lots of small packets", t, func() {
		Convey("running the stream to fill up the klv system", func() {
			Convey("no error is produced and every klv is produced", func() {
				So(noError, ShouldBeNil)
			})
		})
	})

}

// test the good files work as well.
// generate hashes of them as well
// generate a large file for tests this is done after the encoders

type breaker struct {
	file  *os.File
	count *int
}

func (b breaker) Read(in []byte) (int, error) {
	//	fmt.Println(*b.count)
	if *b.count == 1 {
		n, _ := b.file.Read(in)
		return n, io.EOF
	}
	*b.count++

	return b.file.Read(in)
}

type empty struct {
}

func (e empty) Read(in []byte) (int, error) {
	return 0, io.EOF
}

func TestFileBreak(t *testing.T) {

	streamer, _ := os.Open("../../mrx-starter/examples/rexy/rexy_sunbathe_mrx.mxf")
	// wrap the reader in several methods to show the breaking
	/*
		file stream stopping halwaythrough
		bunch of random stuff on the end
		test an empty stream

	*/
	c := 0
	breakError := BufferWrap(breaker{file: streamer, count: &c}, make(chan *KLV, 9000), 10000)

	Convey("Checking that a sudden stop of the stream is handled by the klv", t, func() {
		Convey("running the stream to return an error signalling the stream is incomplete", func() {
			Convey("the error is caught and the stream is stopped", func() {
				So(breakError, ShouldResemble, fmt.Errorf("Buffer stream unexpectantly closed, was expecting at least 218 more bytes"))
			})
		})
	})

	breakError = BufferWrap(empty{}, make(chan *KLV, 9000), 10000)

	Convey("Checking that an empty stream is handled", t, func() {
		Convey("running the buffer stream to immediatly return end of file", func() {
			Convey("the error is caught and the user is notified of an empty data stream", func() {
				So(breakError, ShouldResemble, fmt.Errorf("Empty data stream"))
			})
		})
	})

	// check a non mrx file
}
