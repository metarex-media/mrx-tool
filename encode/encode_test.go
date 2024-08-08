package encode

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/metarex-media/mrx-tool/decode"
	"github.com/metarex-media/mrx-tool/manifest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFileWrite(t *testing.T) {

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/

	testdata := []simpleContents{{key: BinaryClip, contents: [][]byte{[]byte("test metadata")}}}

	simple := simpleTest{contents: testdata, fakeRoundTrip: &manifest.Roundtrip{}}

	writer, newMXR := NewMRXWriterFR("24/1")

	writer.UpdateEncoder(simple)
	fileBuf := bytes.NewBuffer([]byte{})
	err := writer.Encode(fileBuf, &MrxEncodeOptions{})
	_, decodeErr := decode.ExtractStreamData(fileBuf)
	// run the test as if it was being run  by encode, checking each step of the process.
	Convey("Checking that a simple version of the write function works, with a basic set of clipwrapped data", t, func() {
		Convey("checking the write generates an file without error", func() {
			Convey("No error is returned for the encoding", func() {
				So(newMXR, ShouldBeNil)
				So(err, ShouldBeNil)
				So(decodeErr, ShouldBeNil)
			})
		})
	})

	testdataFrame := []simpleContents{{key: TextFrame, contents: [][]byte{[]byte("test metadata"), []byte("test metadata"), []byte("test metadata")}}}

	simpleFrame := simpleTest{contents: testdataFrame, fakeRoundTrip: &manifest.Roundtrip{}}

	writerFrame, newMXRerr := NewMRXWriterFR("24/1")

	writerFrame.UpdateEncoder(simpleFrame)

	fileBufTC := bytes.NewBuffer([]byte{})
	err = writerFrame.Encode(fileBufTC, &MrxEncodeOptions{})
	_, decodeErrTC := decode.ExtractStreamData(fileBufTC)
	// run the test as if it was being run  by encode, checking each step of the process.
	Convey("Checking that a simple version of the write function works, with a basic set of frame wrapped data", t, func() {
		Convey("checking the write generates an file without error, and that the file can be decoded without err", func() {
			Convey("No error is returned for the encoding", func() {
				So(newMXRerr, ShouldBeNil)
				So(err, ShouldBeNil)
				So(decodeErrTC, ShouldBeNil)
			})
		})
	})

	embedAndClips := [][]simpleContents{
		// clip text then frame text
		{{key: TextClip, contents: [][]byte{[]byte("test metadata")}},
			{key: TextFrame, contents: [][]byte{[]byte("test metadata"), []byte("test metadata"), []byte("test metadata")}}},

		// textclip, binary clip, frame binary, text frame
		{{key: TextClip, contents: [][]byte{[]byte("test metadata")}},
			{key: BinaryClip, contents: [][]byte{[]byte("test metadata")}},
			{key: BinaryFrame, contents: [][]byte{[]byte("test metadata"), []byte("test metadata"), []byte("test metadata")}},
			{key: TextFrame, contents: [][]byte{[]byte("test metadata"), []byte("test metadata"), []byte("test metadata")}}},
	}

	// the exepected order of the file handlers
	expectedOrder := [][]string{
		{"060e2b34.01020105.0e090502.01010100", "060e2b34.0101010c.0d01050d.00000000", "060e2b34.01020101.0f020101.05000000"},
		{"060e2b34.01020101.0f020101.01010000", "060e2b34.01020105.0e090502.01010100", "060e2b34.0101010c.0d01050d.00000000", "060e2b34.0101010c.0d01050d.01000000", "060e2b34.01020101.0f020101.05000000"},
	}

	for i, embedAndClip := range embedAndClips {
		embedAndClipFrame := simpleTest{contents: embedAndClip, fakeRoundTrip: &manifest.Roundtrip{}}

		writerembedAndClip, newMXRerr := NewMRXWriterFR("24/1")

		writerembedAndClip.UpdateEncoder(embedAndClipFrame)

		fileBufTCTE := bytes.NewBuffer([]byte{})
		err = writerembedAndClip.Encode(fileBufTCTE, &MrxEncodeOptions{})

		order, decodeErrTCTE := decode.ExtractStreamData(fileBufTCTE)
		// run the test as if it was being run  by encode, checking each step of the process.

		keyOrder := make([]string, len(order))
		for i, key := range order {
			keyOrder[i] = key.MRXID
		}

		Convey("Checking that a simple version of the write function works, with a mix of data types", t, func() {
			Convey("checking the write generates an file without error, and that the file can be decoded without err", func() {
				Convey("No error is returned for the encoding", func() {
					So(newMXRerr, ShouldBeNil)
					So(err, ShouldBeNil)
					So(decodeErrTCTE, ShouldBeNil)
				})
			})
			Convey("The file is written frame data first, then embedded", func() {
				Convey("The key order matches the expected", func() {
					So(keyOrder, ShouldResemble, expectedOrder[i])

				})
			})
		})
	}

	// Checking the manifest
	embedAndClipsManifest := [][]simpleContents{
		// clip text then frame text
		{{key: TextClip, contents: [][]byte{[]byte("test metadata")}},
			{key: TextFrame, contents: [][]byte{[]byte("test metadata"), []byte("test metadata"), []byte("test metadata")}}},
	}

	// the exepected order of the file handlers
	expectedOrderManifest := [][]string{
		{"060e2b34.01020105.0e090502.01010100", "060e2b34.0101010c.0d01050d.00000000", "060e2b34.01020101.0f020101.05000000"},
	}

	b, _ := os.ReadFile("./testdata/base.json")
	var rt manifest.Roundtrip
	json.Unmarshal(b, &rt)
	for i, embedAndClip := range embedAndClipsManifest {
		embedAndClipFrame := simpleTest{contents: embedAndClip, fakeRoundTrip: &rt}

		writerembedAndClip, newMXRerr := NewMRXWriterFR("24/1")

		writerembedAndClip.UpdateEncoder(embedAndClipFrame)

		fileBufTCTE := bytes.NewBuffer([]byte{})
		err = writerembedAndClip.Encode(fileBufTCTE, &MrxEncodeOptions{})

		order, decodeErrTCTE := decode.ExtractStreamData(fileBufTCTE)
		// run the test as if it was being run  by encode, checking each step of the process.

		keyOrder := make([]string, len(order))
		var outConf manifest.Roundtrip
		for i, key := range order {
			keyOrder[i] = key.MRXID

			if key.MRXID == "060e2b34.01020101.0f020101.05000000" {
				json.Unmarshal(key.Data[0], &outConf)
			}
		}

		Convey("Checking that a simple version of the write function works, with a mix of data types and the manifest is preserved", t, func() {
			Convey("checking the write generates an file without error, and that the file can be decoded without err", func() {
				Convey("No error is returned for the encoding", func() {
					So(newMXRerr, ShouldBeNil)
					So(err, ShouldBeNil)
					So(decodeErrTCTE, ShouldBeNil)
				})
			})
			Convey("The file is written frame data first, then embedded", func() {
				Convey("The key order and manifest matches the expected", func() {
					So(keyOrder, ShouldResemble, expectedOrderManifest[i])
					So(rt.Config, ShouldResemble, outConf.Config)
				})
			})
		})
	}
}

// TestFileWrite out of order

type simpleTest struct {
	fakeRoundTrip *manifest.Roundtrip
	contents      []simpleContents
}

type simpleContents struct {
	key      EssenceKey
	contents [][]byte
}

func (st simpleTest) GetRoundTrip() (*manifest.Roundtrip, error) {
	return st.fakeRoundTrip, nil
}

func (st simpleTest) GetStreamInformation() (StreamInformation, error) {

	base := StreamInformation{ChannelCount: len(st.contents), EssenceKeys: make([]EssenceKey, len(st.contents))}

	for i, sc := range st.contents {
		base.EssenceKeys[i] = sc.key
	}

	return base, nil
}

// a simple essence pipe that just puts the data straight through
func (st simpleTest) EssenceChannels(essChan chan *ChannelPackets) error {

	wg := &sync.WaitGroup{}

	for _, datachannel := range st.contents {
		wg.Add(1)
		dataTrain := make(chan *DataCarriage, 10)
		mrxData := ChannelPackets{Packets: dataTrain}

		essChan <- &mrxData
		data := datachannel
		go func() {
			defer wg.Done()
			defer close(dataTrain)

			for _, d := range data.contents {

				deref := d
				dataTrain <- &DataCarriage{Data: &deref, MetaData: &manifest.EssenceProperties{}}
			}
		}()
	}

	wg.Wait()
	return nil
}

func TestUpdateBytes(t *testing.T) {

	// base := Configuration{Version: "any thing"}
	update :=
		manifest.Configuration{
			Version: "0.0.1",
			Default: manifest.StreamProperties{
				FrameRate:  "24/1",
				StreamType: "some data to track",
			},
			StreamProperties: map[int]manifest.StreamProperties{
				1: {
					FrameRate:  "24/1",
					StreamType: "CameraComponent",
				},
				2: {
					FrameRate:  "static",
					StreamType: "Camera Schema",
				},
				3: {
					FrameRate:  "static",
					StreamType: "Static Tail Data",
				},
			},
		}

	base := manifest.Configuration{Version: "0.0.2",
		Default: manifest.StreamProperties{
			FrameRate:  "24/12",
			StreamType: "some thing that is over written",
		}}

	err := configUpdate(&base, update)
	Convey("Checking that the configuration update works", t, func() {
		Convey("updating a base configuration with a large update struct", func() {
			Convey("No error is returned and the base now matches the update function", func() {
				So(err, ShouldBeNil)
				So(base, ShouldResemble, update)
			})
		})
	})
	//	fmt.Println(base, err)

}
