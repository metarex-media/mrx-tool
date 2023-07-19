package encode

import (
	"fmt"
	"io"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileWrite(t *testing.T) {

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/

	testdata := []simpleContents{{key: BinaryClip, contents: [][]byte{[]byte("test metadata")}}}

	simple := simpleTest{contents: testdata, fakeRoundTrip: &Roundtrip{}}

	writer, _ := NewMRXWriterFR("24/1")

	writer.UpdateWriteMethod(simple)

	err := writer.Write(io.Discard, &MrxEncodeOptions{})
	// run the test as if it was being run  by encode, checking each step of the process.
	Convey("Checking that a simple version of the write function works, with a basic set of clipwrapped data", t, func() {
		Convey("checking the write generates an file without error", func() {
			Convey("No error is returned for the encoding", func() {

				So(err, ShouldBeNil)

			})
		})
	})

	testdataFrame := []simpleContents{{key: TextFrame, contents: [][]byte{[]byte("test metadata"), []byte("test metadata"), []byte("test metadata")}}}

	simpleFrame := simpleTest{contents: testdataFrame, fakeRoundTrip: &Roundtrip{}}

	writerFrame, _ := NewMRXWriterFR("24/1")

	writerFrame.UpdateWriteMethod(simpleFrame)

	err = writerFrame.Write(io.Discard, &MrxEncodeOptions{})
	// run the test as if it was being run  by encode, checking each step of the process.
	Convey("Checking that a simple version of the write function works, with a basic set of framewrapped data", t, func() {
		Convey("checking the write generates an file without error", func() {
			Convey("No error is returned for the encoding", func() {

				So(err, ShouldBeNil)

			})
		})
	})

}

type simpleTest struct {
	fakeRoundTrip *Roundtrip
	contents      []simpleContents
}

type simpleContents struct {
	key      EssenceKey
	contents [][]byte
}

func (st simpleTest) GetRoundTrip() (*Roundtrip, error) {
	return st.fakeRoundTrip, nil
}

func (st simpleTest) GetStreamInformation() (StreamInformation, error) {

	base := StreamInformation{ChannelCount: len(st.contents), EssenceKeys: make([]EssenceKey, len(st.contents))}

	for i, sc := range st.contents {
		base.EssenceKeys[i] = sc.key
	}

	return base, nil
}

// a simple essemce pipe that just puts the data straight through
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
				dataTrain <- &DataCarriage{Data: &deref, MetaData: &EssenceProperties{}}
			}
		}()
	}

	wg.Wait()
	return nil
}

func TestUpdateBytes(t *testing.T) {

	//base := Configuration{Version: "any thing"}
	update := `{
        "MrxVersion": 0,
		"Bad":0,
        "DefaultStreamProperties": {
            "FrameRate" : "24/1",
            "Type" : "some data to track"
        },
        "StreamProperties" : {
            "1":{
                "FrameRate" : "24/1",
                "Type" : "CameraComponent"
            },
            "2": {
                "FrameRate"  : "static",
                "Type" : "Camera Schema"
            },
            "3": {
                "FrameRate"  : "static",
                "Type" : "Static Tail Data"
            }
        } }`

	base := Configuration{}

	err := configUpdate(&base, []byte(update))

	fmt.Println(base, err)

}
