package encode

import (
	"io"

	"github.com/metarex-media/mrx-tool/manifest"
)

// EncodeSingleDataStream encodes a single stream of data.
/*

It can be run with the following demo code,
ensuring the channel is closed after writing to finish
the mrx file.

All metadata is given the clocked text key.



	f, _ := os.Create("./testdata/demo.mrx")

	in := make(chan []byte, 10)
	// write the data to the channel
	go func() {
		for i := 0; i < 10; i++ {
			in <- []byte(`{"test":true}`)
		}
		close(in)
	}()

	// set up some default properties
	demoConfig := Configuration{Version: "pre alpha",
		Default:          StreamProperties{StreamType: "some data to track", FrameRate: "24/1", NameSpace: "https://metarex.media/reg/MRX.123.456.789.gps"},
		StreamProperties: map[int]StreamProperties{0: {NameSpace: "MRX.123.456.789.gps"}},
	}

	// run the encoder
	err := EncodeSingleDataStream(f, in, demoConfig)


*/
func EncodeSingleDataStream(destination io.Writer, dataStream chan []byte, streamConfig manifest.Configuration) error { // split this into seperate bits for new calls

	writer := NewMRXWriter()

	input := ExampleFileStream{FakeRoundTrip: &manifest.Roundtrip{Config: streamConfig}, Contents: SingleStream{MdStream: dataStream}}
	writer.UpdateWriteMethod(&input)

	return writer.Write(destination, &MrxEncodeOptions{})
}

// An example for a multi stream encoder

// GetRoundTrip returns the dummy roundtrip
func (st ExampleFileStream) GetRoundTrip() (*manifest.Roundtrip, error) {
	return st.FakeRoundTrip, nil
}

// GetStreamInformation tells the mrx writer that their is one channel
// and that it is of type clocked text.
func (st ExampleFileStream) GetStreamInformation() (StreamInformation, error) {

	base := StreamInformation{ChannelCount: 1, EssenceKeys: []EssenceKey{TextFrame}}

	return base, nil
}

// EssenceChannels is a simple essemce pipe that just puts the data straight through
func (st *ExampleFileStream) EssenceChannels(essChan chan *ChannelPackets) error {

	dataTrain := make(chan *DataCarriage, 10)
	mrxData := ChannelPackets{Packets: dataTrain}

	// send the single channel into the essence channel
	essChan <- &mrxData
	data := st.Contents

	max := len(data.MdStream)
	for i := 0; i < max; i++ {
		d, ok := <-data.MdStream

		if !ok {
			break
		}
		deref := d
		dataTrain <- &DataCarriage{Data: &deref, MetaData: &manifest.EssenceProperties{}}

	}
	// close the channel to stop deadlocks
	close(dataTrain)
	return nil
}

// ExampleFileStream contains the bare minimum
// to get a data stream saved as an MRX.
type ExampleFileStream struct {
	//  A dummy manifest foe examples
	FakeRoundTrip *manifest.Roundtrip

	Contents SingleStream
}

// SingleStream contains a singe metadata channel
// for a data stream
type SingleStream struct {
	//	key      EssenceKey
	MdStream chan []byte
}
