package decode

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileExtract(t *testing.T) {

	files := []string{"../mrx/rexy_sunbathe_mrx.mxf", "./testdata/freeMXF-mxf1.mxf"}

	for _, file := range files {

		streamer, ferr := os.Open(file)
		genErr := EssenceExtractToFile(streamer, "./testdata/essence", false, 4)

		// compare the folder output to the file input
		compErr := compareFolderToFile(streamer, "./testdata/essence", false)

		Convey("Checking that the mrx file can be extracted and saved in a folder", t, func() {
			Convey(fmt.Sprintf("using a %s as the mrx file to decode and extract the data", file), func() {
				Convey("No error is returned", func() {
					So(ferr, ShouldBeNil)
					So(genErr, ShouldBeNil)
				})
			})
			Convey("comparing the contents of the extracted folder to the original file", func() {
				Convey("the contents are identical and were not changed when extracted", func() {
					So(compErr, ShouldBeNil)
				})
			})
		})

		os.RemoveAll("./testdata/essence/")

	}

	for _, file := range files {

		streamer, ferr := os.Open(file)
		genErr := EssenceExtractToFile(streamer, "./testdata/essence/flat", true, 4)

		// compare the folder output to the file input
		compErr := compareFolderToFile(streamer, "./testdata/essence", true)

		Convey("Checking that the mrx file can be extracted and saved in a folder", t, func() {
			Convey(fmt.Sprintf("using a %s as the mrx file to decode and extract the data", file), func() {
				Convey("No error is returned", func() {
					So(ferr, ShouldBeNil)
					So(genErr, ShouldBeNil)
				})
			})
			Convey("comparing the contents of the extracted folder to the original file", func() {
				Convey("the contents are identical and were not changed when extracted", func() {
					So(compErr, ShouldBeNil)
				})
			})
		})
		// delete the files afterwards to prevent pollution
		os.RemoveAll("./testdata/essence/flat")
	}

}

func compareFolderToFile(streamer io.ReadSeeker, foldest string, flat bool) error {

	dirs, _ := os.ReadDir(foldest)
	streamer.Seek(0, 0)
	inputs, _ := ExtractStreamData(streamer)

	if flat {
		base := []*DataFormat{{}}
		for _, in := range inputs {
			base[0].Data = append(base[0].Data, in.Data...)
		}
		inputs = base
	}

	// check simple lengths
	if len(dirs) != len(inputs) {
		return fmt.Errorf("folder contents (%v) and file contents lengths (%v) do not match", len(dirs), len(inputs))
	}

	for i, partition := range inputs {
		nestDirs, _ := filepath.Abs(foldest + "/" + dirs[i].Name())
		nestFiles, _ := os.ReadDir(nestDirs)

		if len(nestFiles) != len(partition.Data) {
			return fmt.Errorf("folder contents of count of %s (%v) and file contents lengths (%v) do not match", nestDirs, len(nestFiles), len(partition.Data))
		}

		// loop through the contents and make a hash of the files
		// they should match
		baseFol := sha256.New()
		baseFile := sha256.New()
		for j, fil := range nestFiles {

			path, _ := filepath.Abs(nestDirs + "/" + fil.Name())

			fb, _ := os.ReadFile(path)
			baseFol.Write(fb)
			baseFile.Write(partition.Data[j])
		}
		folConts := fmt.Sprintf("%x\n", baseFol.Sum(nil))
		fileConts := fmt.Sprintf("%x\n", baseFile.Sum(nil))

		if folConts != fileConts {
			return fmt.Errorf("folder and file contents to not match for partition %v", i)
		}
	}

	return nil
}

func TestStreamEncode(t *testing.T) {

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/
	f, err := os.Open("../encode/testdata/demo.mrx")
	fmt.Println(err)
	_, _ = ExtractStreamData(f)
}
