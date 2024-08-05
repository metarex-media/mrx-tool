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
	mrxFiles := []string{"../testdata/rexy_sunbathe_mrx.mxf", "./testdata/freeMXF-mxf1.mxf", "./testdata/allMdTypes.mrx"}
	outputFile := []string{"./testdata/valid/rexy_sunbathe_mrx.yml", "./testdata/valid/freeMXF-mxf1.yml", "./testdata/valid/allMdTypes.yml"}
	// @TODO have a base yaml to compare to to find the differences

	for i, mrx := range mrxFiles {
		var resultsBuffer bytes.Buffer

		streamer, _ := os.Open(mrx)
		genErr := MRXStructureExtractor(streamer, &resultsBuffer, []int{3, 5, 6, 7, 3}, false)
		// expect the yaml generated to match the hash
		// not have any computational diffrences

		f, _ := os.Create(outputFile[i])
		f.Write(resultsBuffer.Bytes())

		normal, _ := os.ReadFile(outputFile[i])

		htest := sha256.New()
		htest.Write(resultsBuffer.Bytes())
		hnormal := sha256.New()
		hnormal.Write(normal)

		Convey("Checking that the generated yaml of a file structure matches the expected", t, func() {
			Convey(fmt.Sprintf("using a %s as the mrx file to decode and extract the data", mrx), func() {
				Convey(fmt.Sprintf("No error is returned and it matches the expected file of %v", outputFile[i]), func() {
					So(genErr, ShouldBeNil)
					So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, fmt.Sprintf("%x", hnormal.Sum(nil)))
				})
			})
		})

	}

	limits := [][]int{{3}, {1, 2}}

	for i, lmt := range limits {
		var resultsBuffer bytes.Buffer

		streamer, _ := os.Open("../testdata/rexy_sunbathe_mrx.mxf")
		genErr := MRXStructureExtractor(streamer, &resultsBuffer, lmt, false)
		// expect the yaml generated to match the hash
		// not have any computational diffrences

		f, _ := os.Create(outputFile[i])
		f.Write(resultsBuffer.Bytes())

		f2, _ := os.Create(fmt.Sprintf("./testdata/valid/limit%v.yml", i))
		f2.Write(resultsBuffer.Bytes())

		normal, _ := os.ReadFile(fmt.Sprintf("./testdata/valid/limit%v.yml", i))

		htest := sha256.New()
		htest.Write(resultsBuffer.Bytes())
		hnormal := sha256.New()
		hnormal.Write(normal)

		Convey("Checking that the generated yaml of a file structure matches the expected when using different limits", t, func() {
			Convey(fmt.Sprintf("using a %v as the limit to decode and extract the data", lmt), func() {
				Convey("No error is returned and it matches the expected file", func() {
					So(genErr, ShouldBeNil)
					So(fmt.Sprintf("%x", htest.Sum(nil)), ShouldResemble, fmt.Sprintf("%x", hnormal.Sum(nil)))
				})
			})
		})

	}
}

func TestBadFileRead(t *testing.T) {

	// run two different test files with and without index tables
	mrxFiles := []string{"testdata/notanmrx.yaml", "not a file"}
	errors := []string{"Buffer stream unexpectantly closed, was expecting at least 18 more bytes", "error reading and buffering data invalid argument"}
	//	hashes := []string{"4ebf90df1fd10d3cba689f2a313d6c1dc04b23353139ce9441c54d583679b5d6", "e9aa941ee55166c81171f9da12f4b0fcd03b20bbb4dc38fa3b1586ff8e3f4537"}

	for i, mrx := range mrxFiles {
		var resultsBuffer bytes.Buffer

		streamer, _ := os.Open(mrx)
		genErr := MRXStructureExtractor(streamer, &resultsBuffer, []int{3, 5, 6, 7, 3}, false)

		Convey("Checking that the generated yaml of a file matches the expected hash", t, func() {
			Convey(fmt.Sprintf("using a %s as the file to read and extract the data", mrx), func() {
				Convey("No error is returned and the hashes start matching", func() {
					So(genErr, ShouldResemble, fmt.Errorf(errors[i]))

				})
			})
		})

	}

}

// Test bad files e.g. not klv files
