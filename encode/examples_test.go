package encode

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/metarex-media/mrx-tool/decode"
	"github.com/metarex-media/mrx-tool/manifest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMultipleStreams(t *testing.T) {

	// loop through functions that generate lrge data streams to be saved
	testFuncs := []func() (manifest.Configuration, []SingleStream, [][]string, string){
		getMultiplexSetUp, getAll, getEmbed,
	}

	for _, tf := range testFuncs {

		demoConfig, streams, data, mess := tf()

		go func() {
			for i, stream := range streams {
				pos := i
				go func() {
					for _, d := range data[pos] {
						stream.MdStream <- []byte(d)
					}
					// close the stream when the buisness is done
					close(stream.MdStream)
				}()

			}
		}()

		writer, _ := GetMultiStream(streams, &manifest.RoundTrip{Config: demoConfig}) // demoConfig)

		mw := NewMRXWriter()

		mw.UpdateEncoder(writer)
		bufBytes := bytes.NewBuffer([]byte{})
		// 3. run the encoder
		err := mw.Encode(bufBytes, &MrxEncodeOptions{ManifestHistoryCount: 0})

		dec, decErr := decode.ExtractStreamData(bufBytes)

		var nonMatchErr error
		for i, dat := range dec[:len(dec)-2] {

			if len(dat.Data) != len(data[i]) {
				nonMatchErr = fmt.Errorf("the lengths of the %v data stream do not match, got an output of %v and an input of %v", i, len(dat.Data), len(data[i]))
				break
			} else {
				for j, d := range dat.Data {
					if string(d) != data[i][j] {
						nonMatchErr = fmt.Errorf("data point at stream %v pos %v does not match the input", i, j)
						break
					}
				}
			}

			//	fmt.Println(dat.MRXID, dat.FrameRate, string(dat.Data[0]))
		}

		// run the test as if it was being run  by encode, checking each step of the process.
		Convey(mess, t, func() {
			Convey("checking the file is encoded without error and the data is not corrupted", func() {
				Convey("No error is returned for the encoding", func() {

					So(err, ShouldBeNil)
					So(decErr, ShouldBeNil)
					So(nonMatchErr, ShouldBeNil)
				})
			})
		})
	}
}

func getMultiplexSetUp() (demoConfig manifest.Configuration, streams []SingleStream, data [][]string, mess string) {
	demoConfig = manifest.Configuration{Version: "pre alpha",
		Default:          manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1"},
		StreamProperties: map[int]manifest.StreamProperties{2: {FrameRate: "48/1"}},
	}

	bfChannel := make(chan []byte, 5)
	tfChannel := make(chan []byte, 10)
	tf2Channel := make(chan []byte, 5)
	streams = []SingleStream{
		{Key: BinaryFrame, MdStream: bfChannel},
		{Key: TextFrame, MdStream: tfChannel},
		{Key: TextFrame, MdStream: tf2Channel},
	}

	data = [][]string{
		{`{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`},
		{`{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`},
		{`{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`,
			`{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`, `{"test":"text2"}`},
	}
	mess = "Testing multiplexed frame wrapped data, with streams at the same frame rate and the third with a rate twice as fast."

	return
}

func getAll() (demoConfig manifest.Configuration, streams []SingleStream, data [][]string, mess string) {
	demoConfig = manifest.Configuration{Version: "pre alpha",
		Default: manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1"},
	}

	bfChannel := make(chan []byte, 5)
	tfChannel := make(chan []byte, 10)
	beChannel := make(chan []byte, 2)
	teChannel := make(chan []byte, 2)
	streams = []SingleStream{
		{Key: BinaryFrame, MdStream: bfChannel},
		{Key: TextFrame, MdStream: tfChannel},
		{Key: BinaryClip, MdStream: beChannel},
		{Key: TextClip, MdStream: teChannel},
	}

	data = [][]string{
		{`{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`},
		{`{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`},
		{`{"test":"binary embed"}`},
		{`{"test":"text embed"}`},
	}
	mess = "Testing making a file using all four types of data"

	return
}

func getEmbed() (demoConfig manifest.Configuration, streams []SingleStream, data [][]string, mess string) {
	demoConfig = manifest.Configuration{Version: "pre alpha",
		Default: manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1"},
	}

	beChannel := make(chan []byte, 2)
	teChannel := make(chan []byte, 2)
	teChannel2 := make(chan []byte, 2)
	streams = []SingleStream{

		{Key: BinaryClip, MdStream: beChannel},
		{Key: TextClip, MdStream: teChannel},
		{Key: TextClip, MdStream: teChannel2},
	}

	data = [][]string{
		{`{"test":"binary embed"}`},
		{`{"test":"text embed"}`},
		{`{"test":"text embed"}`},
	}
	mess = "Testing making a file using only embedded types of data"

	return
}
