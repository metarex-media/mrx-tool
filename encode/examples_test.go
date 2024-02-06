package encode

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStreamEncode(t *testing.T) {

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/
	f, _ := os.Create("./testdata/demo.mrx")
	in := make(chan []byte, 10)

	go func() {
		for i := 0; i < 10; i++ {
			in <- []byte(`{"test":true}`)
		}
		close(in)
	}()

	demoConfig := Configuration{Version: "pre alpha",
		Default:          StreamProperties{StreamType: "some data to track", FrameRate: "24/1", NameSpace: "https://metarex.media/reg/MRX.123.456.789.gps"},
		StreamProperties: map[int]StreamProperties{0: {NameSpace: "MRX.123.456.789.gps"}},
	}

	err := EncodeSingleDataStream(f, in, demoConfig)

	// run the test as if it was being run  by encode, checking each step of the process.
	Convey("Checking that a simple version of the write function works, with a basic set of clipwrapped data", t, func() {
		Convey("checking the write generates an file without error", func() {
			Convey("No error is returned for the encoding", func() {

				So(err, ShouldBeNil)

			})
		})
	})

	f.Seek(0, 0)

	// decode.ExtractStreamData(f)
}
