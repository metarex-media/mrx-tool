package mrxUnitTest

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/metarex-media/mrx-tool/encode"
	"github.com/metarex-media/mrx-tool/klv"
	"github.com/metarex-media/mrx-tool/manifest"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v3"
)

func TestAST(t *testing.T) {

	mxfToTest := []string{"./testdata/demoReports/goodISXD.mxf",
		"./testdata/demoReports/veryBadISXD.mxf", "./testdata/demoReports/badISXD.mxf"}
	for _, mxf := range mxfToTest {

		doc, docErr := os.Open(mxf)

		klvChan := make(chan *klv.KLV, 1000)

		// generate the AST, assigning the tests
		ast, genErr := MakeAST(doc, klvChan, 10, Specifications{
			Node: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)),
			Part: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode) func(t Test)),
			MXF:  make([]*func(doc io.ReadSeeker, isxdDesc *MXFNode) func(t Test), 0),
		})

		astBytes, yamErr := yaml.Marshal(ast)
		expecBytes, expecErr := os.ReadFile(fmt.Sprintf("%v-ast.yaml", mxf))
		htest := sha256.New()
		htest.Write(astBytes)
		hnormal := sha256.New()
		hnormal.Write(expecBytes)

		Convey("generating a file for testing", t, func() {
			Convey("checking the file is encoded without error and the data is not corrupted", func() {
				Convey("No error is returned for the encoding", func() {

					So(docErr, ShouldBeNil)
					So(genErr, ShouldBeNil)
					So(yamErr, ShouldBeNil)
					So(expecErr, ShouldBeNil)
					So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, fmt.Sprintf("%x", hnormal.Sum(nil)))
				})
			})
		})

	}
}

// TestMakeDemoFiles
// these are example files for testing ISXD
func TestMakeDemoFiles(t *testing.T) {
	// loop through functions that generate lrge data streams to be saved
	testFuncs := []func() (manifest.Configuration, []encode.SingleStream, [][]string, string){
		goodISXD, veryBadISXD, badISXD,
	}

	for _, tf := range testFuncs {

		demoConfig, streams, data, fileName := tf()

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

		f := io.Discard
		var createErr error
		// only make new files if there aren't any saved locally
		if _, err := os.Open(fileName); err != nil {
			f, createErr = os.Create(fileName)

		}

		opts := &encode.MrxEncodeOptions{ManifestHistoryCount: 0}
		if fileName == "./testdata/demoReports/goodISXD.mxf" {
			opts.DisableManifest = true
		}
		err := encode.EncodeMultipleDataStreams(f, streams, demoConfig, opts)

		fread, _ := os.Open(fileName)

		flog, _ := os.Create(fmt.Sprintf("%v.yaml", fileName))
		// _, genErr := MakeAST(f, fout, klvChan, 10)
		//	genErr := ASTTest(f, fout)
		genErr := MRXTest(fread, flog)
		flog.Seek(0, 0)
		fgraw, _ := os.Create(fmt.Sprintf("%v.png", fileName))
		DrawGraph(flog, fgraw)

		// run the test as if it was being run  by encode, checking each step of the process.
		Convey("generating a file for testing", t, func() {
			Convey("checking the file is encoded without error and the data is not corrupted", func() {
				Convey("No error is returned for the encoding", func() {

					So(createErr, ShouldBeNil)
					So(err, ShouldBeNil)
					So(genErr, ShouldBeNil)

				})
			})
		})
	}
}

func goodISXD() (demoConfig manifest.Configuration, streams []encode.SingleStream, data [][]string, mess string) {
	demoConfig = manifest.Configuration{Version: "pre alpha",
		Default: manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1"},
	}

	isxdChannel := make(chan []byte, 5)
	streams = []encode.SingleStream{
		{Key: encode.TextFrame, MdStream: isxdChannel},
	}

	data = [][]string{make([]string, 24)}
	for i := 0; i < 24; i++ {
		fakeData, _ := gofakeit.XML(nil)
		data[0][i] = string(fakeData)
	}
	mess = "./testdata/demoReports/goodISXD.mxf"

	return
}

/*
verybadISXD because:

  - it has multiple keys that aren't ISXD in a framewrapped stream
  - it has embedded data in generic partitions that have different essence keys
  - it has the mrx manifest as a generic partition as well
*/
func veryBadISXD() (demoConfig manifest.Configuration, streams []encode.SingleStream, data [][]string, mess string) {
	demoConfig = manifest.Configuration{Version: "pre alpha",
		Default: manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1"},
	}

	bfChannel := make(chan []byte, 5)
	tfChannel := make(chan []byte, 10)
	beChannel := make(chan []byte, 2)
	teChannel := make(chan []byte, 2)
	streams = []encode.SingleStream{
		{Key: encode.BinaryFrame, MdStream: bfChannel},
		{Key: encode.TextFrame, MdStream: tfChannel},
		{Key: encode.BinaryClip, MdStream: beChannel},
		{Key: encode.TextClip, MdStream: teChannel},
	}

	data = [][]string{
		{`{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`, `{"test":"binary"}`},
		{`{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`, `{"test":"text"}`},
		{`{"test":"binary embed"}`},
		{`{"test":"text embed"}`},
	}
	mess = "./testdata/demoReports/veryBadISXD.mxf"

	return
}

/*
badISXD because it has a MRX manifest that is not part of the isxd spec.
*/
func badISXD() (demoConfig manifest.Configuration, streams []encode.SingleStream, data [][]string, mess string) {

	demoConfig = manifest.Configuration{Version: "pre alpha",
		Default: manifest.StreamProperties{StreamType: "some data to track", FrameRate: "24/1"},
	}

	isxdChannel := make(chan []byte, 5)
	streams = []encode.SingleStream{
		{Key: encode.TextFrame, MdStream: isxdChannel},
	}

	data = [][]string{make([]string, 24)}
	for i := 0; i < 24; i++ {
		fakeData, _ := gofakeit.XML(nil)
		data[0][i] = string(fakeData)
	}
	mess = "./testdata/demoReports/badISXD.mxf"

	return
}
