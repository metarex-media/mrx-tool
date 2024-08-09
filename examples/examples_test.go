package examples

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExamples(t *testing.T) {

	err := MultiStream()
	Convey("Checking the multistream example is running", t, func() {
		Convey("runnning the multistream example", func() {
			Convey("no error is produced and the mrx file is produced", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	os.Remove("./testdata/demo.mrx")

	fsErr := FolderScan()
	Convey("Checking the folderscan example is running", t, func() {
		Convey("runnning the folderscan example", func() {
			Convey("no error is produced and the mrx file is produced", func() {
				So(fsErr, ShouldBeNil)
			})
		})
	})

	os.Remove("./testdata/folderscan.mrx")
}
