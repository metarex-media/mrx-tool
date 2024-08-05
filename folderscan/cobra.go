package folderscan

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/metarex-media/mrx-tool/encode"
	"github.com/metarex-media/mrx-tool/manifest"
	"github.com/spf13/cobra"
)

var encodeIn string
var encodeOut string
var encodeFrameRate string
var encodeManifestCount int
var overWrite string

func init() {
	// set up flags for the two different decode commands
	EncodeCmd.Flags().StringVar(&encodeIn, "input", "", "identifies the  input folder to be encoded")
	EncodeCmd.Flags().StringVar(&encodeOut, "output", "", "the name of the file to be generated")
	EncodeCmd.Flags().StringVar(&encodeFrameRate, "framerate", "", "gives the frame rate of the video in the form x/y e.g. 29.97 fps is 30000/1001")
	EncodeCmd.Flags().IntVar(&encodeManifestCount, "previousManifest", 0, "The count of previous manifests to be included in the manifest, from 0 upwards. -1 is show all")
	EncodeCmd.Flags().StringVar(&overWrite, "overwrite", "", "a json string to overwrite some or all of the configuration file")

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

The folders represent partitions within the mrx file and are to be numerically ordered, with their numbers. e.g. 1 is represented as 0001
Each partition folder, then contains up to 9999 essence files.

The essence names are as follows:
- TC - Clocked text data
- BE - Embedded timing binary data
- BC - Clocked binary data
- TE - Embedded timing text data

The complete folder names would then look like 0000StreamTC or
0000StreamBC etc.

The metadata stored within the folders is then named
{number}d, e.g. 25d, 132341d or 01d would all be valid.

The numerical order the data is found in the folders
is the order it is saved in the file.

Any manifest files that are found can be added onto the "previous" field for a new manifest that is generated.
The manifest, carries all the metadata in the mrx file, allowing for optional data from privates sources.
These manifest files are stored as config.json in the parent folder.

Flat formats are also used where the metadata is not split up into folders,
and instead the data stream is part of the name. e.g. 0000StreamTC01d

`,

	// Run interactively unless told to be batch / server
	RunE: Encode,
}

// Encode encodes the scanned contents of a folder(s) as an MRX files
func Encode(_ *cobra.Command, _ []string) error {

	// check the input file was given

	err := inoutCheck(encodeIn, encodeOut)
	if err != nil {
		return err
	}

	var mw *encode.MrxWriter

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

	var update manifest.Configuration
	if overWrite != "" {
		err = json.Unmarshal([]byte(overWrite), &update)
		if err != nil {
			return fmt.Errorf("error parsing \"%s\" : %v", overWrite, err)
		}
	}

	writeMethod := &FolderScanner{ParentFolder: encodeIn}
	mw.UpdateWriteMethod(writeMethod)
	err = mw.Write(f, &encode.MrxEncodeOptions{ManifestHistoryCount: encodeManifestCount, ConfigOverWrite: update})

	if err != nil {
		return err
	}

	fmt.Printf("%v has been generated \n", encodeOut)

	return nil
}
