package decode

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/metarex-media/mrx-tool/klv"
	"golang.org/x/sync/errgroup"
)

// EssenceExtractToFile extracts the contents from an MRX and dumps it to a folder as individual
// files, where each file is an individual metadata entry from the file.
func EssenceExtractToFile(stream io.Reader, parentFolder string, flat bool, leadingZeros int) error {

	klvChan := make(chan *klv.KLV, 1000)
	parentFolder, _ = filepath.Abs(parentFolder)

	// check if the folder exits
	_, err := os.Stat(parentFolder)
	if os.IsNotExist(err) {
		err = os.MkdirAll(parentFolder, os.ModePerm)

		if err != nil {
			return fmt.Errorf("error generating destination folder %v", err)
		}
	}

	return essenceExtractToFile(stream, klvChan, parentFolder, flat, leadingZeros)
}

type essenceSaveTarget struct {
	parentStream int
	parentFolder string
	partition    string
	// partitionCount int
	// move along with the folder
	// rolling partition count
	essenceCount int
}

type mrxPartitionPosition struct {
	dataStreams           map[essID]*essenceSaveTarget
	currentPartitionCount int
	currentPartitionName  string

	nextDataStreamCount int
}

// essID contains the properties for an essence key to be unique
type essID struct {
	key string
	sid int
}

/*
type pos struct {
	part     int
	par      string
	essCount int
}*/

// essencExtract extracts just the essence and its keys from themMRX
func essenceExtract(stream io.Reader, buffer chan *klv.KLV) ([]*DataFormat, error) {

	outData := make([]*DataFormat, 0)

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream
	errs.Go(func() error {
		return klv.StartKLVStream(stream, buffer, 10)

	})

	//	location := essenceFolder{parentFolder: parentFolder, streamCount: map[int]int{}}
	location := mrxPartitionPosition{dataStreams: make(map[essID]*essenceSaveTarget)}
	// initiate the klv handling stream
	errs.Go(func() error {

		// clean out the channel at the end
		// this is to prevent channel deadlocks further down the chain
		defer func() {
			_, klvOpen := <-buffer
			for klvOpen {
				_, klvOpen = <-buffer
			}
		}()

		// get the first bit of stream
		klvItem, klvOpen := <-buffer

		// handle each klv packet
		for klvOpen {

			// check if it is a partition key
			// if not its presumed to be essence
			if partitionName(klvItem.Key) == "060e2b34.020501  .0d010201.01    00" {

				// decode the partition
				err := location.partitionDecode(klvItem, buffer)

				if err != nil {

					return err
				}

			} else {

				// decode as essence
				//

				err := location.essenceWrite(&outData, klvItem)
				if err != nil {

					return err
				}
			}

			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}
		return nil
	})

	// wait for routines then handle the error
	// if there is an error.
	err := errs.Wait()

	if err != nil {
		return nil, err
	}

	// if everything has been read end the extraction
	return outData, nil
}

// essenceExtractToFile takes and mrx file stream and decodes the data streams into seperate folders/files.
func essenceExtractToFile(stream io.Reader, buffer chan *klv.KLV, parentFolder string, flat bool, leadingZeros int) error {

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream
	errs.Go(func() error {
		return klv.StartKLVStream(stream, buffer, 10)

	})

	//	location := essenceFolder{parentFolder: parentFolder, streamCount: map[int]int{}}
	location := mrxPartitionPosition{dataStreams: make(map[essID]*essenceSaveTarget)}
	// initiate the klv handling stream
	errs.Go(func() error {

		// clean out the channel at the end
		// this is to prevent channel deadlocks further down the chain
		defer func() {
			_, klvOpen := <-buffer
			for klvOpen {
				_, klvOpen = <-buffer
			}
		}()

		// get the first bit of stream
		klvItem, klvOpen := <-buffer

		// handle each klv packet
		for klvOpen {

			// check if it is a partition key
			// if not its presumed to be essence
			if partitionName(klvItem.Key) == "060e2b34.020501  .0d010201.01    00" {

				// decode the partition
				err := location.partitionDecode(klvItem, buffer)

				if err != nil {

					return err
				}

			} else {

				// decode as essence
				var err error
				if flat {
					err = location.essenceSaveFlat(parentFolder, klvItem, leadingZeros)
				} else {
					err = location.essenceSave(parentFolder, klvItem, leadingZeros)
				}

				if err != nil {

					return err
				}
			}

			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}
		return nil
	})

	// wait for routines then handle the error
	// if there is an error.
	err := errs.Wait()

	if err != nil {
		return err
	}
	// if everything has been read end the extraction
	return nil
}

func (e *mrxPartitionPosition) partitionDecode(klvItem *klv.KLV, metadata chan *klv.KLV) error {
	// /	e.essenceCount = 0
	//	shift, lengthlength := klvItem
	partitionLayout := partitionExtract(klvItem)

	e.currentPartitionCount = int(partitionLayout.BodySID)

	// update the current partition layout location
	e.currentPartitionName = partitionLayout.PartitionType
	if e.currentPartitionName == "body" {
		e.currentPartitionName = "mrxip"

	}
	// flush out the header metadata
	// as it is not used yet (apart from the primer)
	flushedMeta := 0
	for flushedMeta < int(partitionLayout.HeaderByteCount) {
		flush, open := <-metadata

		if !open {
			return fmt.Errorf("error when using klv data klv stream interrupted")
		}
		flushedMeta += flush.TotalLength()

	}

	// hoover up the indextable and remove it to rpevent it being mistaken as essence
	if partitionLayout.IndexTable {
		_, open := <-metadata
		if !open {
			return fmt.Errorf("error when using klv data klv stream interrupted")
		}
	}
	// position += md.currentContainer.HeaderLength

	return nil
}

var pathSeparator = string(os.PathSeparator)

func (e *mrxPartitionPosition) essenceWrite(layouter *[]*DataFormat, data *klv.KLV) error {

	// derefernce for donig sums
	layout := *layouter
	// get the positional information

	writeTarget := e.getCounter(string(data.Key))
	// generate the file name as a flat path
	pos := writeTarget.parentStream

	/*
		essLabel := essLabeller(data.Key)

		if essLabel == "manifest" {
			outData := make([][]byte, 1)
			outData[0] = data.Value
			layout = append(layout, &DataFormat{Data: outData})

			var mid manifest.Roundtrip
			json.Unmarshal(data.Value, &mid)

			for stream, properties := range mid.Config.StreamProperties {

				layout[stream].MRXID = properties.NameSpace
			}

			*layouter = layout
			return nil
		}*/

	// stop it adding up etc
	if len(layout) < pos+1 {
		outData := make([][]byte, 0)
		layout = append(layout, &DataFormat{Data: outData, MRXID: fullName(data.Key)})

	}

	midData := layout[pos].Data
	midData = append(midData, data.Value)

	layout[pos].Data = midData

	*layouter = layout

	writeTarget.increment()

	return nil
}

func (e *mrxPartitionPosition) essenceSaveFlat(parentFolder string, data *klv.KLV, leadingZeros int) error {

	// get the positional information

	writeTarget := e.getCounter(string(data.Key))
	// generate the file name as a flat path

	essLabel := essLabeller(data.Key)

	if essLabel == "manifest" {
		return manifestSave(parentFolder+pathSeparator, data)
	}

	basePath := fmt.Sprintf(parentFolder+pathSeparator+"%04dStream%s", writeTarget.parentStream, essLabel)
	//	basePath += fmt.Sprintf("%04d"+writeTarget.partition, writeTarget.partitionCount)
	essFile, err := os.Create(basePath + leadingZero(writeTarget.essenceCount, leadingZeros) + "d")

	if err != nil {
		return err
	}

	_, err = essFile.Write(data.Value)

	if err != nil {
		return err
	}

	writeTarget.increment()

	return nil
}

func manifestSave(basePath string, data *klv.KLV) error {
	essFile, err := os.Create(basePath + "config.json")

	if err != nil {
		return err
	}

	_, err = essFile.Write(data.Value)

	return err

}

func (e *mrxPartitionPosition) essenceSave(parentFolder string, data *klv.KLV, leadingZeros int) error {

	writeTarget := e.getCounter(string(data.Key))
	essLabel := essLabeller(data.Key)

	if essLabel == "manifest" {
		return manifestSave(parentFolder+pathSeparator, data)
	}

	// check for mnaifest before saving

	basePath := fmt.Sprintf(parentFolder+pathSeparator+"%04dStream%s", writeTarget.parentStream, essLabel)

	if _, err := os.Stat(basePath); os.IsNotExist(err) {

		err := os.MkdirAll(basePath, 0777)

		if err != nil {
			return err
		}
	}

	// e.partitionCount = e.streamCount[e.parentStream]

	/*	basePath += fmt.Sprintf(pathSeparator+"%04d"+writeTarget.partition, writeTarget.partitionCount)

		if _, err := os.Stat(basePath); os.IsNotExist(err) {

			err := os.Mkdir(basePath, 0777)

			if err != nil {
				return err
			}
		}*/

	essFile, err := os.Create(basePath + pathSeparator + leadingZero(writeTarget.essenceCount, leadingZeros) + "d")

	if err != nil {
		return err
	}

	_, err = essFile.Write(data.Value)

	if err != nil {
		return err
	}

	writeTarget.increment()

	return nil
}

func (e *essenceSaveTarget) increment() {
	e.essenceCount++
}

func leadingZero(num int, zeroLength int) string {

	numberString := fmt.Sprintf("%d", num)
	zeroDiff := zeroLength - len(numberString)

	for zeroDiff > 0 {
		numberString = "0" + numberString
		zeroDiff--
	}

	return numberString
}

// getCounter returns the folder information for a unique data stream
// if the stream is new then the information is generated from the current partition information
func (e *mrxPartitionPosition) getCounter(key string) *essenceSaveTarget {
	writeTarget, ok := e.dataStreams[essID{key: key, sid: e.currentPartitionCount}]

	if !ok {

		writeTarget = &essenceSaveTarget{parentStream: e.nextDataStreamCount,
			parentFolder: e.currentPartitionName, partition: e.currentPartitionName}

		e.dataStreams[essID{key: key, sid: e.currentPartitionCount}] = writeTarget
		e.nextDataStreamCount++
	}

	return writeTarget

}

/*
var textFrameKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x01, 0x01, 0x01})
var binaryClipKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x03, 0x00, 0x00})
var textClipKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x04, 0x00, 0x00})
var binaryFrameKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00})
var manifestKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x05, 0x00, 0x00})*/

var textFrameKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x7f, 0x01, 0x7f})

var binaryFrameKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x7f, 0x00, 0x7f})
var textClipKey = string([]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0000, 0x7f, 0, 0x7f})
var binaryClipKey = string([]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0001, 0x7f, 0, 0x7f})
var manifestKey = string([]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x05, 0x7f, 0x00, 0x7f})

func essLabeller(key []byte) string {

	// mask the key
	key[13], key[15] = 0x7f, 0x7f

	switch string(key) {

	case textFrameKey:
		return "TC"
	case binaryClipKey:
		return "BE"
	case textClipKey:
		return "TE"
	case binaryFrameKey:

		return "BC"
	case manifestKey:
		return "manifest"
	default:

		return "essence"
	}

}
