package examples

import (
	"os"

	"github.com/metarex-media/mrx-tool/encode"
	"github.com/metarex-media/mrx-tool/folderscan"
)

func FolderScan() error {
	mw := encode.NewMRXWriter()
	// create the mrx
	f, err := os.Create("./testdata/folderscan.mrx")
	if err != nil {
		return err
	}

	// choose a folder to write to
	writeMethod := &folderscan.FolderScanner{ParentFolder: "./testdata/flat"}
	mw.UpdateEncoder(writeMethod)

	// run the mrx writer
	return mw.Encode(f, &encode.MrxEncodeOptions{})
}
