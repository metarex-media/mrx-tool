package mrxUnitTest

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/metarex-media/mrx-tool/encode"
	"github.com/metarex-media/mrx-tool/manifest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAST(t *testing.T) {

	// run two different test files with and without index tables
	mrxFiles := []string{"../testdata/rexy_sunbathe_mrx.mxf", "./testdata/all.mxf", "../tmp/ISXD.mxf"}
	// hashes := []string{"4ebf90df1fd10d3cba689f2a313d6c1dc04b23353139ce9441c54d583679b5d6", "e9aa941ee55166c81171f9da12f4b0fcd03b20bbb4dc38fa3b1586ff8e3f4537"}

	for _, mrx := range mrxFiles {
		//	var resultsBuffer bytes.Buffer
		// fmt.Println(i)
		//	streamer, _ := os.Open(mrx)
		f, _ := os.Open(mrx)
		//	klvChan := make(chan *klv.KLV, 1000)
		//		fout, _ := os.Create(fmt.Sprintf("tester%v.yaml", i))
		//	flog, _ := os.Create(fmt.Sprintf("tester%v.yaml", i))
		// _, genErr := MakeAST(f, fout, klvChan, 10)
		//	genErr := ASTTest(f, fout)
		genErr := MRXTest(f, io.Discard)
		// expect the yaml generated to match the hash
		// not have any computational diffrences

		// htest := sha256.New()
		// htest.Write(resultsBuffer.Bytes())
		Convey("Checking that the generated yaml of a file matches the expected hash", t, func() {
			Convey(fmt.Sprintf("using a %s as the file to read and extract the data", mrx), func() {
				Convey("No error is returned and the hashes start matching", func() {
					So(genErr, ShouldBeNil)
					// So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, hashes[i])
				})
			})
		})

	}

}

// TestMakeDemoFiles
func TestMakeDemoFiles(t *testing.T) {
	// loop through functions that generate lrge data streams to be saved
	testFuncs := []func() (manifest.Configuration, []encode.SingleStream, [][]string, string){
		goodISXD, getAll, getEmbed,
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
		f, createErr := os.Create(fileName)
		opts := &encode.MrxEncodeOptions{ManifestHistoryCount: 0}
		if fileName == "./testdata/demoReports/goodISXD.mxf" {
			opts.DisableManifest = true
		}
		err := encode.EncodeMultipleDataStreams(f, streams, demoConfig, opts)
		f.Close()

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

func getAll() (demoConfig manifest.Configuration, streams []encode.SingleStream, data [][]string, mess string) {
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

func getEmbed() (demoConfig manifest.Configuration, streams []encode.SingleStream, data [][]string, mess string) {
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
