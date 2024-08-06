package encode

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/metarex-media/mrx-tool/decode"
	"github.com/metarex-media/mrx-tool/manifest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStreamEncode(t *testing.T) {

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/

	in := make(chan []byte, 10)

	// fil the channel then close it
	for i := 0; i < 10; i++ {
		in <- []byte(`{"test":true}`)
	}
	close(in)

	demoConfig := manifest.Configuration{Version: "pre alpha",
		Default:          manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1", NameSpace: "https://metarex.media/reg/MRX.123.456.789.gps"},
		StreamProperties: map[int]manifest.StreamProperties{0: {NameSpace: "MRX.123.456.789.gps"}},
	}

	bufBytes := bytes.NewBuffer([]byte{})
	err := EncodeSingleDataStream(bufBytes, in, demoConfig)
	dec, decErr := decode.ExtractStreamData(bufBytes)

	var nonMatchErr error

	// set up a test to prevent the manifest coming out first
	var d *decode.DataFormat
	for _, dat := range dec {
		if dat.MRXID != "060e2b34.01020101.0f020101.05000000" {
			d = dat
		}
		fmt.Println(dat.MRXID, dat.FrameRate, string(dat.Data[0]))
	}
	fmt.Println(len(dec), err, decErr, nonMatchErr)
	if d != nil {
		for _, data := range d.Data {
			if string(data) != `{"test":true}` {
				nonMatchErr = fmt.Errorf("data not sent got %s instead of {\"test\":true}", string(data))
			}
		}
	}
	// run the test as if it was being run  by encode, checking each step of the process.
	Convey("Checking that a simple version of the write function works, with a basic set of clipwrapped data", t, func() {
		Convey("checking the write generates an file without error", func() {
			Convey("No error is returned for the encoding", func() {
				So(d, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(decErr, ShouldBeNil)
				So(nonMatchErr, ShouldBeNil)
			})
		})
	})

	//
}
