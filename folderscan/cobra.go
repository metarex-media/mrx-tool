package folderscan

import (
	"fmt"
	"os"

	"github.com/metarex-media/mrx-tool/encode"
	"github.com/spf13/cobra"
)

var encodeIn string
var encodeOut string
var encodeFrameRate string
var encodeManifestCount int

func init() {
	//set up flags for the two different decode commands
	EncodeCmd.Flags().StringVar(&encodeIn, "input", "", "identifies the  input folder to be encoded")
	EncodeCmd.Flags().StringVar(&encodeOut, "output", "", "the name of the file to be generated")
	EncodeCmd.Flags().StringVar(&encodeFrameRate, "framerate", "", "gives the frame rate of the video in the form x/y e.g. 29.97 fps is 30000/1001")
	EncodeCmd.Flags().IntVar(&encodeManifestCount, "previousManifest", 0, "The count of previous manifests to be included in the manifest, from 0 upwards. -1 is show all")

}

func inoutCheck(in, out string) error {
	if in == "" {
		return fmt.Errorf("no input file chosen please use the --input flag")
	}

	if out == "" {
		return fmt.Errorf("no output destination chosen please use the --output flag")
	}

	return nil
}

var EncodeCmd = &cobra.Command{
	Use:   "encode",
	Short: "Encode an organised folder of metadata into an mrx file",
	Long: `The encode flag brings together an organised file system of metadata into an mrx file.
detailing the labels of its contents and the overall file structure

The folders represent partitions within the mrx file and are to be numerically ordered, with 4 digit numbers. e.g. 1 is represented as 0001
Each partition folder, then contains up to 9999 essence files.

The essence names are as follows:
- frameText - frame text data
- clipBin - clip binary data
- clipText - clip test data
- frameBin - frame binary data

The order the data is found is the order it is saved in the file.

Any manifest files that are found can be added onto the "previous" field for a new manifest that is generated.
The manifest, carries all the metadata in the mrx file, allowing for optional data from privates sources.

`,

	// Run interactively unless told to be batch / server
	RunE: DecodeRun,
}

func DecodeRun(Command *cobra.Command, args []string) error {

	//check the input file was given

	err := inoutCheck(encodeIn, encodeOut)
	if err != nil {
		return err
	}

	var mw *encode.MxfWriter

	if encodeFrameRate == "" { // the framerate key isn't really required yet
		mw = encode.NewMRXWriter()
	} else {
		mw, err = encode.NewMRXWriterFR(encodeFrameRate)
	}

	if err != nil {
		return err
	}

	f, err := os.Create(encodeOut)
	if err != nil {
		return err
	}

	writeMethod := &folderScanner{folder: encodeIn}
	mw.UpdateWriteMethod(writeMethod)
	err = mw.Write(f, &encode.MrxEncodeOptions{ManifestHistoryCount: encodeManifestCount})

	if err != nil {
		return err
	}

	fmt.Printf("%v has been generated \n", encodeOut)

	return nil
}
