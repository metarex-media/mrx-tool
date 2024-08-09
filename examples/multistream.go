package examples

import (
	"os"

	"github.com/metarex-media/mrx-tool/encode"
	"github.com/metarex-media/mrx-tool/manifest"
)

// MultiStream is an example of how to write multiple streams of data
// using the encode library. This function can imported or
// copy and pasted to be played around with.
func MultiStream() error {

	// set up some channels
	bfChannel := make(chan []byte, 5)
	tfChannel := make(chan []byte, 10)

	// set up the streams
	streams := []encode.SingleStream{
		{Key: encode.BinaryFrame, MdStream: bfChannel},
		{Key: encode.TextFrame, MdStream: tfChannel}}

	// set up your data
	data := [][]string{
		{`{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`},
		{`{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`}}

	f, err := os.Create("./testdata/demo.mrx")
	if err != nil {
		return err
	}

	// fill up the channels as a go function
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

	// set up some default properties
	demoConfig := manifest.Configuration{Version: "pre alpha",
		Default:          manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1", NameSpace: "https://metarex.media/reg/MRX.123.456.789.gps"},
		StreamProperties: map[int]manifest.StreamProperties{0: {NameSpace: "MRX.123.456.789.gps"}},
	}

	// run the encoder
	return encode.EncodeMultipleDataStreams(f, streams, demoConfig, nil)

}
