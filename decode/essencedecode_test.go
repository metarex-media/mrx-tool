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
