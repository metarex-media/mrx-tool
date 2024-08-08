package encode

import (
	"context"
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/manifest"
	"golang.org/x/sync/errgroup"
)

// SingleStream contains a singe metadata channel
// for a data stream
type SingleStream struct {
	Key      EssenceKey
	MdStream chan []byte
}

// GetRoundTrip returns the roundtrip file
func (st ExampleMultipleStream) GetRoundTrip() (*manifest.RoundTrip, error) {
	return st.RoundTrip, nil
}

// GetStreamInformation tells the mrx writer the channel keys for the metadata.
// The number of keys is the number of channels.
func (st ExampleMultipleStream) GetStreamInformation() (StreamInformation, error) {

	if st.StreamInfo == nil {
		return StreamInformation{}, fmt.Errorf("no stream information found, ensure the stream is initialised")
	}

	return *st.StreamInfo, nil
}

// EssenceChannels is a pipe that concurrently
// runs all the metadata streams at once.
func (st *ExampleMultipleStream) EssenceChannels(essChan chan *ChannelPackets) error {

	// use errs to handle errors while running concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream

	for i, stream := range st.Contents {
		// set up the stream outside of the concurrent loop to preserve order
		pos := i
		dataTrain := make(chan *DataCarriage, 10)
		mrxData := ChannelPackets{Packets: dataTrain, OverViewData: manifest.GroupProperties{StreamID: pos}}
		essChan <- &mrxData

		errs.Go(func() error {
			// close the channel to stop deadlocks
			defer close(dataTrain)

			d, ok := <-stream.MdStream
			for ok {

				deref := d
				dataTrain <- &DataCarriage{Data: &deref, MetaData: &manifest.EssenceProperties{}}
				d, ok = <-stream.MdStream
			}

			return nil
		})
	}
	// close the channel to stop deadlocks

	return errs.Wait()
}

// ExampleFileStream contains the bare minimum
// to get multiple data streams saved as an MRX.
type ExampleMultipleStream struct {
	//  The roundtrip associated with the streams
	RoundTrip *manifest.RoundTrip
	// The array of essence keys for the metadata streams
	StreamInfo *StreamInformation
	// The metadata data stream channels to handle
	Contents []SingleStream
}

// GetMultiStream returns a multistream encode object.
// This can then be used to update teh encoder properties
// of an *MRXWriter object.
// Like so:
//
//	mw.UpdateEncoder(writer)
func GetMultiStream(streams []SingleStream, roundTrip *manifest.RoundTrip) (*ExampleMultipleStream, error) {

	StreamInf := StreamInformation{EssenceKeys: make([]EssenceKey, len(streams))}

	for i, s := range streams {
		if s.Key == 0 {
			return nil, fmt.Errorf("undeclared key for channel x, please ensure the essence keys is chosen")
		}

		// @TODO check the data key is present
		StreamInf.EssenceKeys[i] = s.Key
	}

	return &ExampleMultipleStream{Contents: streams, StreamInfo: &StreamInf, RoundTrip: roundTrip}, nil
}

// EncodeSMultipleDataStreams encodes multiple streams of metadata as an MRX.
/*

It can be run with the following demo code,
ensuring the channel is closed after writing to finish
the mrx file.

	// set up some channels
	bfChannel := make(chan []byte, 5)
	tfChannel := make(chan []byte, 10)

	// set up the streams
	streams = []encode.SingleStream{
		{Key: encode.BinaryFrame, MdStream: bfChannel},
		{Key: encode.TextFrame, MdStream: tfChannel}}

	// set up your data
	data = [][]string{
		{`{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`},
		{`{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`}}

	f, _ := os.Create("./testdata/demo.mrx")

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
		Default:          StreamProperties{StreamType: "some data to track", FrameRate: "24/1", NameSpace: "https://metarex.media/reg/MRX.123.456.789.gps"},
		StreamProperties: map[int]StreamProperties{0: {NameSpace: "MRX.123.456.789.gps"}},
	}

	// run the encoder
	err := encode.EncodeMultipleDataStreams(f, in, demoConfig)


*/
func EncodeMultipleDataStreams(destination io.Writer, streams []SingleStream, streamConfig manifest.Configuration, encodeOptions *MrxEncodeOptions) error { // split this into seperate bits for new calls

	writer, err := GetMultiStream(streams, &manifest.RoundTrip{Config: streamConfig}) // demoConfig)

	if err != nil {
		return err
	}

	mw := NewMRXWriter()

	mw.UpdateEncoder(writer)

	return mw.Encode(destination, encodeOptions)
}
