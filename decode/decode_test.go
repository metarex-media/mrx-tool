package decode

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileRead(t *testing.T) {

	// run two different test files with and without index tables
	mrxFiles := []string{"../testdata/rexy_sunbathe_mrx.mxf", "../testdata/Disney_Test_Patterns_ISXD.mxf"}
	hashes := []string{"4ebf90df1fd10d3cba689f2a313d6c1dc04b23353139ce9441c54d583679b5d6", "e9aa941ee55166c81171f9da12f4b0fcd03b20bbb4dc38fa3b1586ff8e3f4537"}

	for i, mrx := range mrxFiles {
		var resultsBuffer bytes.Buffer

		streamer, _ := os.Open(mrx)
		genErr := StreamDecode(streamer, &resultsBuffer, []int{3, 5, 6, 7, 3}, false)
		// expect the yaml generated to match the hash
		// not have any computational diffrences

		htest := sha256.New()
		htest.Write(resultsBuffer.Bytes())
		Convey("Checking that the generated yaml of a file matches the expected hash", t, func() {
			Convey(fmt.Sprintf("using a %s as the file to read and extract the data", mrx), func() {
				Convey("No error is returned and the hashes start matching", func() {
					So(genErr, ShouldBeNil)
					So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, hashes[i])
				})
			})
		})

	}
}

func TestBadFileRead(t *testing.T) {

	// run two different test files with and without index tables
	mrxFiles := []string{"testdata/notanmrx.yaml"}
	//	hashes := []string{"4ebf90df1fd10d3cba689f2a313d6c1dc04b23353139ce9441c54d583679b5d6", "e9aa941ee55166c81171f9da12f4b0fcd03b20bbb4dc38fa3b1586ff8e3f4537"}

	for _, mrx := range mrxFiles {
		var resultsBuffer bytes.Buffer

		streamer, _ := os.Open(mrx)
		genErr := StreamDecode(streamer, &resultsBuffer, []int{3, 5, 6, 7, 3}, false)
		// expect the yaml generated to match the hash
		// not have any computational diffrences
		htest := sha256.New()
		htest.Write(resultsBuffer.Bytes())
		Convey("Checking that the generated yaml of a file matches the expected hash", t, func() {
			Convey(fmt.Sprintf("using a %s as the file to read and extract the data", mrx), func() {
				Convey("No error is returned and the hashes start matching", func() {
					So(genErr, ShouldResemble, fmt.Errorf("Invalid MRX File"))
					//		So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, hashes[i])
				})
			})
		})

	}
}

// Test bad files e.g. not klv files

/*

objects to test:
generating the correct outouts

handling index tables
testing individual klv values and seeing what happens

*/
