package encode

import (
	"encoding/json"
	"io"
)

// EncodeSingleDataStream encodes a single stream of data.
/*

Setting up a stream can be done lick so.
Closing the channel signals to the writer to finish
the writing the mrx file



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


*/
func EncodeSingleDataStream(destination io.Writer, dataStream chan []byte, streamConfig Configuration) error { // split this into seperate bits for new calls

	writer := NewMRXWriter()

	conf := `{
				"MrxVersion": "pre alpha",
				"DefaultStreamProperties": {
					"FrameRate": "24/1",
					"Type": "some data to track",
					"NameSpace": "https://metarex.media/reg/MRX.123.456.789.gps"
				},
				"StreamProperties": {
					"0": {
						"NameSpace": "https://metarex.media/ui/reg/MRX.123.456.789.gps/register.json"
					}
				}
			}`

	var enConf Configuration
	json.Unmarshal([]byte(conf), &enConf)

	input := fileStream{fakeRoundTrip: &Roundtrip{Config: streamConfig}, contents: singleStreamContents{contents: dataStream}}
	writer.UpdateWriteMethod(&input)

	writer.Write(destination, &MrxEncodeOptions{})

	return nil
}

// An example for a multi stream encoder

func (st fileStream) GetRoundTrip() (*Roundtrip, error) {
	return st.fakeRoundTrip, nil
}

func (st fileStream) GetStreamInformation() (StreamInformation, error) {

	base := StreamInformation{ChannelCount: 1, EssenceKeys: []EssenceKey{TextFrame}}

	return base, nil
}

// a simple essemce pipe that just puts the data straight through
func (st *fileStream) EssenceChannels(essChan chan *ChannelPackets) error {

	dataTrain := make(chan *DataCarriage, 10)
	mrxData := ChannelPackets{Packets: dataTrain}

	essChan <- &mrxData
	data := st.contents

	max := len(data.contents)
	for i := 0; i < max; i++ {
		d, ok := <-data.contents

		if !ok {
			break
		}
		deref := d
		dataTrain <- &DataCarriage{Data: &deref, MetaData: &EssenceProperties{}}

	}
	// close the channlel tos top deadlocks
	close(dataTrain)
	return nil
}

type fileStream struct {
	fakeRoundTrip *Roundtrip
	contents      singleStreamContents
}

type singleStreamContents struct {
	key      EssenceKey
	contents chan []byte
}
