package decode

import (
	"fmt"
	"os"
	"testing"
	//. "github.com/smartystreets/goconvey/convey"
)

func TestFileExtract(t *testing.T) {

	// empty, _ := os.Create("../result/helpme.yaml")
	//stream, _ := os.Open("../mrx-starter/examples/newtests/Disney_Test_Patterns_ISXD.mxf")
	streamer, _ := os.Open("../mrx/rexy_sunbathe_mrx.mxf")
	fmt.Println(EssenceDecode(streamer, "./testdata", false, 4))

	os.Mkdir("./testdata/flat", 0777)
	streamer, _ = os.Open("../mrx/rexy_sunbathe_mrx.mxf")
	fmt.Println(EssenceDecode(streamer, "./testdata/flat", true, 4))
}

func TestStreamEncode(t *testing.T) {

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/
	f, err := os.Open("../encode/testdata/demo.mrx")
	fmt.Println(err)
	ExtractStreamData(f)
}
