// Package folderscan handles the folder scanning and encoding methods
package folderscan

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/metarex-media/mrx-tool/encode"
	"github.com/metarex-media/mrx-tool/manifest"
	"golang.org/x/sync/errgroup"
)

// FolderScanner checks the folder for an MRX layout.
// It enables the encode.Writer interface to save
// the folder contents as an MRX.
type FolderScanner struct {
	ParentFolder string
	FolLayout    *fullFolderMRX
}

// GetStreamInformation finds the number of channels and their MRX keys to be saved
func (f *FolderScanner) GetStreamInformation() (encode.StreamInformation, error) {
	folderLayout, err := folderScan(f.ParentFolder)

	if err != nil {
		return encode.StreamInformation{}, err
	}

	// essenceKeys := folderLayout.foundEssence

	f.FolLayout = &folderLayout
	// fmt.Println(f)

	// essenceKeys
	keys := orderKeys(folderLayout.streams)
	essenceKeys := make([]encode.EssenceKey, len(folderLayout.streams))
	for i, k := range keys {
		essenceKeys[i] = folderLayout.streams[k].partitionType
	}

	return encode.StreamInformation{EssenceKeys: essenceKeys}, nil

}

// GetRoundTrip gets the configuration and a manifest.
// It searches the parent folder for a config.json file,
// if the file is not found then it is not used.
// The config.json must be of type encode.Roundtrip
func (f *FolderScanner) GetRoundTrip() (*manifest.RoundTrip, error) {

	var configBody manifest.RoundTrip

	roundBytes, err := os.ReadFile(f.ParentFolder + osSeperator + "config.json")

	if err == nil {

		err := json.Unmarshal(roundBytes, &configBody)
		return &configBody, err
	}

	return &manifest.RoundTrip{}, nil
}

// EssenceChannels extracts the essence from the files, it then sends one data
// stream (in numerical order) to the writer channel.
func (f *FolderScanner) EssenceChannels(essChan chan *encode.ChannelPackets) error {

	// close the channels once they've been written to
	defer close(essChan)

	keys := orderKeys(f.FolLayout.streams)
	errs, _ := errgroup.WithContext(context.Background())
	//	for _, partition := range f.flay.folders {

	// loop through the folders
	for keyPos := range keys {

		streamKey := keys[keyPos]

		dataTrain := make(chan *encode.DataCarriage, 10)
		mrxData := encode.ChannelPackets{Packets: dataTrain}

		essChan <- &mrxData

		errs.Go(func() error {

			stream := f.FolLayout.streams[streamKey]
			// pKeys := orderKeys(stream.contents)

			err := func() error {

				// sent := false
				// set up partiton packet
				// only use the partition packe tif manifest data is not found

				defer func() {
					// cose the data once the writing has finished
					close(dataTrain)
				}()

				for i := 0; i <= stream.max; i++ {
					// for _, pKey := range pKeys {

					ess, ok := stream.contents[i]

					// @TODO handle the manifest in a new way
					commonInformation := manifest.GroupProperties{}
					//	pcKeys := orderKeys(partition.contents)

					//	for _, pcKey := range pcKeys {
					// ess := partition.contents[pcKey]

					// extract the klvs

					var carriage *encode.DataCarriage
					var err error

					// if data has been missed out form the folder then empty data is sent
					// so that the frame placement of the data is preserved.
					if !ok {
						carriage = &encode.DataCarriage{}
					} else {
						carriage, err = essExtract(ess.fullLocation)
					}
					// fmt.Println(err)
					if err != nil {
						return err
					}
					commonInformation.StreamType = stream.partitionTypeHuman

					dataTrain <- carriage
					mrxData.OverViewData = commonInformation

				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
	}

	return errs.Wait()

}

func orderKeys[T any](long map[int]T) []int {
	keys := make([]int, len(long))
	i := 0
	for position := range long {
		keys[i] = position
		i++
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	return keys
}

// essExtract extracts the data, along with any accompanying metadata.
func essExtract(essenceFile string) (*encode.DataCarriage, error) {

	essFile, err := os.Open(essenceFile)
	if err != nil {
		return nil, fmt.Errorf("error extracting data to encode from %v:%v", essenceFile, err)
	}

	fInfo, err := essFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("error extracting file information from %v:%v", essenceFile, err)
	}

	essData := make([]byte, fInfo.Size())
	_, err = essFile.Read(essData)
	if err != nil {
		return nil, fmt.Errorf("error extracting data to encode from %v:%v", essenceFile, err)
	}

	has := sha256.New()
	has.Write(essData)

	// dataKLV := klv.KLV{Key: []byte(folderType), Length: length, Value: }
	metadata := manifest.EssenceProperties{EditDate: fInfo.ModTime().String(), Hash: fmt.Sprintf("%64x", has.Sum(nil)), DataOrigin: essenceFile}

	return &encode.DataCarriage{Data: &essData, MetaData: &metadata}, nil
}

// the folder is built up of a map of streams, partitions then their essence
// all of which should have unique values
type fullFolderMRX struct {
	streams      map[int]*partition // []folderMRX
	foundEssence []encode.EssenceKey
}

type partition struct {
	// partition can span multiple partitions
	// and keeps the data type fot the stream
	partitionType      encode.EssenceKey
	partitionTypeHuman string
	// @TODO update essence container to map[int]map[int]essenceMRX
	contents map[int]essenceMRX
	// Max is the maximum document position
	// this is used for the decoding later
	max int
}

/*type folderMRX struct {
	// contents map[int]essenceMRX
	contents map[int]essenceMRX
}*/

type essenceMRX struct {
	key          encode.EssenceKey
	fullLocation string
}

// folder order rege
var allBody = regexp.MustCompile(`^\d{1,}d$`)
var streamFol = regexp.MustCompile(`^\d{4}stream((tc)|(te)|(bc)|(be))$`)

// var flatBodyStructure = regexp.MustCompile(`^\d{4}stream\d{4}((mrxip)|(header))`)
var flatBodyStructure = regexp.MustCompile(`^\d{4}stream((tc)|(te)|(bc)|(be))\d{1,}d`)

// osSeperator stringfys the os.Separator to prevent repetetive code
var osSeperator = string(os.PathSeparator)

// folderScan returns the folders and contents that contain essence to be
// wrapped when an mrx file is generated
func folderScan(folder string) (fullFolderMRX, error) {
	folder, _ = filepath.Abs(folder)
	folders, err := os.ReadDir(folder)

	if err != nil {
		return fullFolderMRX{}, fmt.Errorf("error reading folder %v : %v", folder, err)
	}

	folderLayout := fullFolderMRX{streams: make(map[int]*partition)}

	for _, fold := range folders {

		if fold.IsDir() {

			// extract the folder
			err := folderExtract(fold, &folderLayout, folder)

			if err != nil {
				return fullFolderMRX{}, err
			}

		} else {
			// estract the flat file systems
			err := fileExtract(fold, &folderLayout, folder)
			if err != nil {
				return fullFolderMRX{}, err
			}
		}

	}

	return folderLayout, nil
}

func fileExtract(fold fs.DirEntry, essenceFile *fullFolderMRX, parentFolder string) error {

	folname := strings.ToLower(fold.Name())

	if flatBodyStructure.MatchString(folname) {
		streamPos := 0

		_, err := fmt.Sscanf(folname, "%dstream", &streamPos)

		if err != nil {
			return fmt.Errorf("error extracting stream position from folder %s: %v", folname, err)
		}

		essKey, essString := essKeyTypeExtract(fold.Name()[10:12])

		// prevent nil errors in the stream layout
		if _, ok := essenceFile.streams[streamPos]; !ok {
			essenceFile.streams[streamPos] = &partition{contents: make(map[int]essenceMRX), partitionType: essKey, partitionTypeHuman: essString}
			// only add the file type on the fist version
			essenceFile.foundEssence = append(essenceFile.foundEssence, essKey)
		}

		//	essKey, essString := essKeyTypeExtract(fold.Name()[essSplit:])

		// just skip for the moment if the keys are not found
		if essKey == 0 {
			return nil
		}

		// check that the type of essence is as expected
		if essenceFile.streams[streamPos].partitionType == 0 {
			essenceFile.streams[streamPos].partitionType = essKey
		} else if essenceFile.streams[streamPos].partitionType != essKey {
			return fmt.Errorf("mixed essence file types found in %v, please ensure they are all the same type", parentFolder)
		}

		ess := essenceMRX{fullLocation: parentFolder + osSeperator + fold.Name(), key: essKey}
		// get the essence position
		essencePos := 0
		_, err = fmt.Sscanf(folname[12:], "%dd", &essencePos)
		if err != nil {
			return fmt.Errorf("error extracting essence position from file %s: %v", folname, err)
		}
		// @TODO check for duplicate essence as a safety barrier

		if essencePos > essenceFile.streams[streamPos].max {
			essenceFile.streams[streamPos].max = essencePos
		}

		essenceFile.streams[streamPos].contents[essencePos] = ess

	}

	return nil
}

func folderExtract(fold fs.DirEntry, essenceFolder *fullFolderMRX, parentFolder string) error {

	folname := strings.ToLower(fold.Name())

	if streamFol.MatchString(folname) {
		streamPos := 0
		_, err := fmt.Sscanf(folname, "%04dstream", &streamPos)
		if err != nil {
			return fmt.Errorf("error extracting stream position from folder %s: %v", folname, err)
		}

		key, humanKey := essKeyTypeExtract(folname[10:])

		if _, ok := essenceFolder.streams[streamPos]; !ok {
			essenceFolder.streams[streamPos] = &partition{contents: make(map[int]essenceMRX), partitionType: key, partitionTypeHuman: humanKey}
		}
		strFol := parentFolder + osSeperator + fold.Name()
		streamFolders, err := os.ReadDir(strFol)
		if err != nil {
			return fmt.Errorf("error reading folder %v : %v", parentFolder, err)
		}

		// ASSIGN the information here

		for _, strFile := range streamFolders {

			strName := strFile.Name()
			if allBody.MatchString(strName) { // bodyFol.MatchString(folname) || headerFol.MatchString(folname) {
				filFol := strFol + osSeperator + strFile.Name()

				contentPosition := 0
				_, err := fmt.Sscanf(strFile.Name(), "%dd", &contentPosition)
				if err != nil {
					return fmt.Errorf("error extracting essence position from file %s: %v", folname, err)
				}

				if contentPosition > essenceFolder.streams[streamPos].max {
					essenceFolder.streams[streamPos].max = contentPosition
				}

				essenceFolder.streams[streamPos].contents[contentPosition] = essenceMRX{fullLocation: filFol, key: key}

				// check if the essence key has already been added

				//	}

			}
		}
		// essenceFolder.foundEssence = append(essenceFolder.foundEssence, essKey)

	}

	return nil
}

func essKeyTypeExtract(folName string) (encode.EssenceKey, string) {
	switch folName {
	case "TC", "tc", "Tc", "tC":

		return encode.TextFrame, "Text based frame data"
	case "BE", "be", "Be", "bE":

		return encode.BinaryClip, "Binary based clip data"
	case "TE", "te", "Te", "tE":

		return encode.TextClip, "Text based clip data"
	case "BC", "bc", "Bc", "bC":

		return encode.BinaryFrame, "Binary based frame data"
	default:
		// move to the next >
		return 0, ""
	}
}

/*
func foldEssScan(folder string, foundFiles *folderMRX) (encode.EssenceKey, string, error) {
	folder, _ = filepath.Abs(folder)
	folders, err := os.ReadDir(folder)

	if err != nil {
		return 0, "", fmt.Errorf("error reading folder %v : %v", folder, err)
	}

	var folderHuman string
	var folderType encode.EssenceKey
	//	foundFiles := folderMRX{contents: map[int]essenceMRX{}}

	for _, fold := range folders {

		if !fold.IsDir() {
			folName := fold.Name()
			location := folder + osSeperator + folName

			var key encode.EssenceKey
			key, folderHuman = essKeyTypeExtract(folName)
			if key == 0 {
				continue
			}

			ess := essenceMRX{key: key, fullLocation: location}

			//ess.fullLocation = location

			if folderType == 0 {
				folderType = ess.key
			} else if folderType != ess.key {
				return 0, "", fmt.Errorf("Mixed essence file types found in %v, please ensure they are all the same type", folder)
			}

			essencePos := 0

			fmt.Sscanf(folName[0:4], "%04d", &essencePos)
			// match the name to give a key and full path
			// if not picked up ignroe for the moment

			// check the value is not being redeclared at this point

			foundFiles.contents[essencePos] = ess
			//match to the essence types
		}
	}

	if folderType == 0 {
		return 0, "", fmt.Errorf("unidentified essence in folder")
	}

	return folderType, folderHuman, nil
}*/
