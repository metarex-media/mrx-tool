package decode

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var decodeIn string
var decodeOut string
var decodeSplit string
var jsonFile bool

var decodeSaveIn string
var decodeSaveOut string
var zeroCount int

func init() {
	//set up flags for the two different decode commands
	DecodeCmd.Flags().StringVar(&decodeIn, "input", "", "identifies the file to be decoded")
	DecodeCmd.Flags().StringVar(&decodeOut, "output", "", "the file to be generated and the decode infomration to be saved to")
	DecodeCmd.Flags().StringVar(&decodeSplit, "split", "", "split gives an input")
	DecodeCmd.Flags().BoolVar(&jsonFile, "json", false, "a flag for the output format to be json, instead of the default yaml.")

	DecodeSaveCmd.Flags().StringVar(&decodeSaveIn, "input", "", "identifies the file to be decoded")
	DecodeSaveCmd.Flags().StringVar(&decodeSaveOut, "output", "", "the base folder for the seperated essence to be saved into")
	DecodeSaveCmd.Flags().IntVar(&zeroCount, "leadingZeroCount", 4, "the minimum integer length of the saved files")

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

var DecodeCmd = &cobra.Command{
	Use:   "decode",
	Short: "Decode an mrx file structure into yaml form",
	Long: `The decode flag breaks down the selected mrx file into a yaml file, 
detailing the labels of its contents and the overall file structure


The yaml contains an array of partitions and their essence information in the order they were found in the mrx file. 
The file has the following fields.


The partition section contains the following information
- Partition Type identifies the essence container. e.g. Header or Body
- HeaderLength is the length of the header and any metadata it may contain.
- EssenceByteCount is the total byte length of all the essence.
- ContentPackageCount is the number of individual content packages within the partition.
- IndexTable indetifies if a index table is present in this partition showing some of the data contained within it.
- ContentPackages is an array of the contentpackage found in the partition, in the order it was found in the partition. Each conent package is an essence array.
- Warning provides a string stating any potential issues within the essence.
- skipped content is an object stating how many content packages were not included and their total byte count.
- ContentPackageStatistics contains the average, variance and standard deviation in the lengths of the content packages, as well as the longest and shortest package.


The essence array contains the following information in each element. It has the following fields.
- Key, this is the UL of the container.
- Symbol, this is the identifier of the essence type.
- Description, the description of the essence as found in the smpte register, or auto generated information where the key was not identified.
- File Offset the offset in the file for the start of the data **NOT** the start of the essence container.
- length is the length of the essence data
- Type is the resolved container key if it can be found.
- TotalByteCount is the total count of the essence including the UL and BER encoded length Bytes.  

content packages with the key "00000000.00000000.00000000.00000000" are skipped content packages, these represent an array of content packages as a single item.	`,

	// Run interactively unless told to be batch / server
	RunE: DecodeRun,
}

func DecodeRun(Command *cobra.Command, args []string) error {

	//check the input file was given
	if decodeIn == "" {
		return fmt.Errorf("no input file chosen please use the --input flag")
	}

	decodespl, err := decodeSpliter(decodeSplit)
	if err != nil {
		return err
	}

	// do some error checking
	decodeIn, _ := filepath.Abs(decodeIn)

	f, err := os.Open(decodeIn)
	if err != nil {
		return fmt.Errorf("Error reading %v: %v", decodeIn, err)
	}

	// check if the outwriter is a stream or straight to stdout
	var fout io.Writer
	if decodeOut != "" {
		decodeOut, _ := filepath.Abs(decodeOut)
		fout, err = os.Create(decodeOut)
		if err != nil {
			return fmt.Errorf("Error generating the output file %v: %v", decodeIn, err)
		}
	} else {
		fout = os.Stdout
	}

	err = StreamDecode(f, fout, decodespl, jsonFile)
	if err != nil {
		return err
	}

	// if not writing to stdout tell the user the service has run
	if fout != os.Stdout {
		fmt.Println("Written to", decodeOut)
	}

	return nil
}

// decodeSplitter converts a string of 1,2,3,4 into an array of []int{1,2,3,4}
func decodeSpliter(splitString string) ([]int, error) {

	if splitString == "" {
		return []int{}, nil
	}

	// replace any spaces as a back up
	splitString = strings.ReplaceAll(splitString, " ", "")

	splits := strings.Split(splitString, ",")
	splitSlice := make([]int, len(splits))

	for i, s := range splits {

		size, err := strconv.Atoi(s)

		if err != nil {
			return []int{}, fmt.Errorf("Error encouterd splitting %v: %v", splitString, err)
		}

		splitSlice[i] = size

	}

	return splitSlice, nil
}

// running 26 gb files - have a footer and a header saved in testdata
// generate a body partition and fill with 50mb things

var DecodeSaveCmd = &cobra.Command{
	Use:   "decodesave",
	Short: "Extract the essence of an mrx file and saves each essence as an individual file.",
	Long: `An mrx decoder that extracts the essence of a file and saves each essence as an individual file.

Within in the output folder, child folders named after the partition type are generated, e.g. 0000Body. Within each folder
the essence is saved in the order it is found. Under the naming prefix of 0000{{esskeytype}}.

The esskeyTypes are:
- frameText - frame text data
- clipBin - clip binary data
- clipText - clip test data
- frameBin - frame binary data

	`,

	// Run interactively unless told to be batch / server
	RunE: DecodeSaveRun,
}

// DecodeSaveRun is the function called to extract the esence
func DecodeSaveRun(Command *cobra.Command, args []string) error {

	err := inoutCheck(decodeSaveIn, decodeSaveOut)
	if err != nil {
		return err
	}

	f, err := os.Open(decodeSaveIn)
	if err != nil {
		return err
	}

	if zeroCount < 0 || zeroCount > 8 {
		return fmt.Errorf("The leadingZeroCount of %d is invalid please choose a number between 0 and 8", zeroCount)
	}

	err = EssenceDecode(f, decodeSaveOut, false, zeroCount)

	if err != nil {
		return err
	}

	fmt.Println("Written to", decodeSaveOut)

	return nil

}
