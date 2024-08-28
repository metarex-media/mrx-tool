// Package encode is for encoding mrx files.
// It contains a generic interface for you to include your own data inputs for mxf files
package encode

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/metarex-media/mrx-tool/manifest"

	mxf2go "github.com/metarex-media/mxf-to-go"
	"github.com/peterbourgon/mergemap"
	"golang.org/x/sync/errgroup"
)

// The Encoder interface is a way to plug in the essence generator into an MRX file to save
// the essence, in an generic way without having to deal with the MRX file internal layout
// and data, such as headers.
type Encoder interface {

	// GetEssenceKeys gives the array of the essence Keys
	// to be used in this mrx file.
	// The essence keys are given in the order their
	// metadata channels are handled in the EssenceChannels
	// function.
	GetStreamInformation() (StreamInformation, error)

	// The essence Pipe returns streams of KLV and their associated metadata
	// Each channel represents a separate stream. These streams
	// are written to MRX following the MRX file rules.
	EssenceChannels(chan *ChannelPackets) error

	// RoundTrip gets the RoundTrip data associated with the metadata
	// stream, this is an optional piece of metadata.
	GetRoundTrip() (*manifest.RoundTrip, error)
}

// ChannelPackets contains the user metadata for a metadata stream
// and the channel that is fed the metadata stream.
type ChannelPackets struct {
	OverViewData manifest.GroupProperties
	Packets      chan *DataCarriage
}

// DataCarriage contains the metadata bytes
// and any metametadata associated with it.
type DataCarriage struct {
	Data     *[]byte
	MetaData *manifest.EssenceProperties
}

// StreamInformation contains the information
// about the complete metadata stream.
type StreamInformation struct {
	// ChannelCount is the number of channels
	// ChannelCount int

	// Essence Keys are the essence keys of each data type
	// in the order they are to be streamed to the encoder
	// It also is the total number of channels expected
	EssenceKeys []EssenceKey
}

// MrxEncodeOptions are the encoding parameters.
type MrxEncodeOptions struct {
	// ManifestHistoryCount is
	// the number of previous manifest files to be inlcuded (if possible)
	ManifestHistoryCount int
	// ConfigOverwrite overwrites any fields in the base configuration
	// of the mrx file. e.g from previous manifests
	ConfigOverWrite manifest.Configuration
	// is the manifest file to be used
	// default is to include it
	DisableManifest bool
}

// Encode writes the data to an mrx file, default options are used if MrxEncodeOptions is nil
func (mw *MrxWriter) Encode(w io.Writer, encodeOptions *MrxEncodeOptions) error {

	// get the mrxWriter methods
	mrxwriter := mw.saver

	if mrxwriter == nil {
		return fmt.Errorf("error saving, no essence extraction methods available")
	}

	if encodeOptions == nil {
		encodeOptions = &MrxEncodeOptions{}
	}

	// Header data set up
	essenceStream, err := mrxwriter.GetStreamInformation()
	if err != nil {
		return err
	}

	round, err := mw.saver.GetRoundTrip()
	if err != nil {
		return err
	}

	// this is where the config update would come in
	err = configUpdate(&round.Config, encodeOptions.ConfigOverWrite)

	if err != nil {
		return err
	}

	// merge the user options and the parsed information
	cleanStream, err := streamClean(essenceStream, round.Config)
	if err != nil {
		return fmt.Errorf("error configuring essence %v", err)
	}

	// get the essence keys
	containerKeys := cleanStream.containerKeys
	// set the writer infromation
	// @TODO check its the clean stream stuff
	mw.frameInformation.StreamTimeLine = round.Config

	// generate the UMDID for this mrx file
	mw.uMIDFinish(len(containerKeys))

	if !encodeOptions.DisableManifest {
		// trim the manifest off to prevent errors occuring
		cleanStream.manifest = true
	}

	// metadata set up
	headerMeta := mw.metaData(cleanStream)

	// have an object to reference how far through the file generation we are as we generate it
	filePosition := &partitionPosition{partitions: []RIPLayout{}, totalByteCount: 0, prevPartition: 0}

	// write the header partition
	err = writePartition(w, filePosition, headerName(header, false, false), 0, headerMeta, containerKeys)
	if err != nil {
		return err
	}

	// encode the essence and get the manifest information
	manifesters, err := encodeEssence(w, filePosition, mrxwriter, cleanStream)
	if err != nil {
		return err
	}

	// generate the manifest and core, encoding to
	manifestBytes, err := mw.encodeRoundTrip(round, manifesters, cleanStream, encodeOptions.ManifestHistoryCount)
	if err != nil {
		return err
	}

	if !encodeOptions.DisableManifest {
		// write the manifest and update the position
		err = writePartition(w, filePosition, headerName(genericStream, false, false), 0, []byte{}, containerKeys)
		if err != nil {
			return err
		}

		_, err = w.Write(manifestBytes)
		if err != nil {
			return fmt.Errorf("error writing manifest %v", err)
		}

		filePosition.totalByteCount += len(manifestBytes)
	}

	// check or essence extraction error handling
	// set the SID back to  0 at the end, then write the footer
	filePosition.sID = 0
	err = writePartition(w, filePosition, headerName(footer, true, true), uint64(filePosition.totalByteCount), headerMeta, containerKeys)
	if err != nil {
		return err
	}

	// Finally the RIP
	_, err = w.Write(rIPPack(filePosition.partitions))

	if err != nil {
		return fmt.Errorf("error writing Random Index Pack %v", err)
	}

	return nil
}

// essence filter contains the properties of an stream
type channelProperties struct {
	clocked         bool
	key             []byte
	frameRate       mxf2go.TRational
	frameMultiplier int
	nameSpace       string
}

type mrxLayout struct {
	dataStreams   []channelProperties
	containerKeys [][]byte
	baseFrameRate mxf2go.TRational
	//
	isxdflag bool
	// reorder flags is framewrapped data is declared after clip wrapped
	// so that the config can be reorderd when it is saved as part of the mxf file
	// for roundtripping
	reorder  bool
	manifest bool
}

// stream clean goes through the esesnce
// updating to match the infomration the user provided and sotring out the container
func streamClean(foundStream StreamInformation, userStream manifest.Configuration) (mrxLayout, error) {

	var fullStream mrxLayout
	var base bool
	var clip bool

	cleanEssence := make([]channelProperties, len(foundStream.EssenceKeys))

	keyTypes := make(map[EssenceKey]int)
	containers := make(map[string]bool)

	for i, baseKey := range foundStream.EssenceKeys {
		// map starts counting from 1, the array starts from  0

		// handle the essence key, generating its bytes for frame wrapping
		// and assign the container key
		essenceKey := getKeyBytes(baseKey)
		containers[string(getContainerKey(baseKey))] = true
		if baseKey == TextFrame {
			fullStream.isxdflag = true
		}

		// update the name if its a frame wrapped key
		if baseKey == TextFrame || baseKey == BinaryFrame {

			if clip {
				fullStream.reorder = true
			}

			cleanEssence[i].clocked = true
			startPos := keyTypes[baseKey]

			essenceKey[15] = byte(startPos)
			keyTypes[baseKey]++

			// extract frame rate here
			getFrame := userStream.StreamProperties[i].FrameRate
			if getFrame == "" {
				getFrame = userStream.Default.FrameRate

			}

			var num, dom int32
			if getFrame != "" {
				_, err := fmt.Sscanf(getFrame, "%v/%v", &num, &dom)

				if err != nil {
					return mrxLayout{}, fmt.Errorf("error finding framerate %v", err)
				}
			}

			if num == 0 || dom == 0 {
				// @TODO implement a better way to handle thos
				num, dom = 24, 1
				fullStream.baseFrameRate = mxf2go.TRational{Numerator: 24, Denominator: 1}
			}

			if !base {

				fullStream.baseFrameRate = mxf2go.TRational{Numerator: num, Denominator: dom}
				base = true
				// only one item in this content package as it is the frame rate essence
				cleanEssence[i].frameMultiplier = 1
				//	essenceKey[13] = 1
			} else {

				// @TODO include some error handling for when its greater than 127
				// or less than 1
				cleanEssence[i].frameMultiplier = int(num) / int(fullStream.baseFrameRate.Numerator)
				// essenceKey[13] = byte(int(num) / int(fullStream.baseFrameRate.Numerator))
			}
			essenceKey[13] = byte(cleanEssence[i].frameMultiplier)
			cleanEssence[i].frameRate = mxf2go.TRational{Numerator: num, Denominator: dom}

		} else {
			// else thr static files are default properies
			// clip is used to signal if the channels configuration will have to be reordered
			clip = true
		}

		cleanEssence[i].nameSpace = userStream.StreamProperties[i].NameSpace
		cleanEssence[i].key = essenceKey

	}

	// generate the container keys after every input has been checked
	for c := range containers {
		fullStream.containerKeys = append(fullStream.containerKeys, []byte(c))

	}

	/*
		if data is going to be reorderd fix the stream
	*/
	fullStream.dataStreams = cleanEssence

	return fullStream, nil
}

func configUpdate(base *manifest.Configuration, overWrite manifest.Configuration) error {

	updateBytes, err := json.Marshal(overWrite)
	if err != nil {
		return fmt.Errorf("error handling the configuration overwrite struct: %v", err)
	}

	var update map[string]any
	err = json.Unmarshal(updateBytes, &update)
	if err != nil {
		return fmt.Errorf("error handling the configuration overwrite struct: %v", err)
	}

	baseToMap, err := json.Marshal(base)
	if err != nil {
		return fmt.Errorf("error handling the configuration base struct: %v", err)
	}
	baseMap := make(map[string]any)
	err = json.Unmarshal(baseToMap, &baseMap)

	if err != nil {
		return fmt.Errorf("error setting up the base configuration to merge: %v", err)
	}

	merged := mergemap.Merge(baseMap, update)

	combinedBytes, err := json.Marshal(merged)

	if err != nil {
		return fmt.Errorf("error getting complete configuration struct: %v", err)
	}

	// json.Unmarshal(combinedBytes, base)

	err = json.Unmarshal(combinedBytes, base)

	if err != nil {
		return fmt.Errorf("error updating the configuration with the merged changes: %v", err)
	}

	return nil
}

type partitionPosition struct {
	partitions     []RIPLayout
	totalByteCount int
	prevPartition  int
	sID            int
}

func writePartition(w io.Writer, filePosition *partitionPosition, header [16]byte, footerPos uint64, headerMeta []byte, essenceKeys [][]byte) error {
	// get the length of the partition
	partitionLength := 108 - 20 + len(essenceKeys)*16
	// generate the partition bytes
	partition := partitionPack{Signature: header, SizeKAG: 1, HeaderByteCount: uint64(len(headerMeta)), PartitionLength: partitionLength, PreviousPartition: uint64(filePosition.prevPartition),
		FooterPartition: footerPos, MajorVersion: 1, MinorVersion: 3, ThisPartition: uint64(filePosition.totalByteCount), BodySID: uint32(filePosition.sID)}
	partitionBytes, _ := encodePartition(partition, essenceKeys)

	// update the RIP pack with the position
	filePosition.partitions = append(filePosition.partitions, RIPLayout{SID: uint32(filePosition.sID), partitionPosition: uint64(filePosition.totalByteCount)})
	// update the previous partition, then move along the total position along
	filePosition.prevPartition = filePosition.totalByteCount
	filePosition.totalByteCount += len(partitionBytes) // partitionLength
	filePosition.totalByteCount += len(headerMeta)

	// write the partition information
	_, err := w.Write(partitionBytes)
	if err != nil {
		return fmt.Errorf("error writing partition %v", err)
	}
	_, err = w.Write(headerMeta)

	if err != nil {
		return fmt.Errorf("error writing header %v", err)
	}

	return nil
}

func encodeEssence(w io.Writer, filePosition *partitionPosition, mrxwriter Encoder, essSetup mrxLayout) ([]manifest.Overview, error) {
	// set up the partition channels, generating as many channels as there are streams
	essenceContainers := make(chan *ChannelPackets, len(essSetup.dataStreams))

	clockCount := 0

	// get the clcoked essence positions

	for _, baseKey := range essSetup.dataStreams {

		if baseKey.clocked {

			clockCount++
		}

	}

	// essenceKeys := essSetup.EssenceKeys
	// use errs to handle errors while running concurrently
	// this is to allow us to use the channels
	errs, _ := errgroup.WithContext(context.Background())
	// initiate the klv stream
	errs.Go(func() error {
		return mrxwriter.EssenceChannels(essenceContainers)
	})

	// do some stream set up establishing th ekeys

	type pipeWrapper struct {
		pack *ChannelPackets
		info channelProperties
	}

	clockDataStreams := make([]*pipeWrapper, clockCount)
	unClockDataStreams := make([]*pipeWrapper, len(essSetup.dataStreams)-clockCount)

	var clockPos, unClockPos int

	// set up the datastreams from the input

	for _, set := range essSetup.dataStreams {

		essPipe := <-essenceContainers

		if set.clocked {
			clockDataStreams[clockPos] = &pipeWrapper{pack: essPipe, info: set}
			clockPos++
		} else {

			unClockDataStreams[unClockPos] = &pipeWrapper{pack: essPipe, info: set}
			unClockPos++
		}

	}

	// fmt.Println(unClockDataStreams, clockDataStreams)
	// set up the mainfest information holders
	manifesters := make([]manifest.Overview, len(essSetup.dataStreams))
	filePosition.sID = 1
	partitionManifest := []manifest.Overview{}

	// set up a stream flag
	availableEssence := true

	if len(clockDataStreams) == 0 {
		availableEssence = false
	}

	// multiplex the framewrapped data together
	if availableEssence {
		err := writePartition(w, filePosition, headerName(body, false, false), 0, []byte{}, essSetup.containerKeys)
		if err != nil {
			return nil, err
		}
	}

	for availableEssence {

		for i, pipe := range clockDataStreams {

			// to stop extra data being written after the first channel is closed
			for j := 0; j < pipe.info.frameMultiplier; j++ {

				essPacket, essChanOpen := <-pipe.pack.Packets

				if !essChanOpen && i == 0 {

					// only stop the stream when the key data has finished being sent
					availableEssence = false
					continue
				}

				// if the main channel has closed hoover the remaining channels to prevent go deadlocking
				if !availableEssence {
					for essChanOpen {
						_, essChanOpen = <-pipe.pack.Packets
					}
					continue
				}

				if !essChanOpen {
					essPacket = &DataCarriage{Data: &[]byte{}}
				} else {
					man := essPacket.MetaData

					manifesters[i].Essence = append(manifesters[i].Essence, *man)
					manifesters[i].Common = pipe.pack.OverViewData
				}
				essKLV := essPacket.Data
				berLength := mxf2go.BEREncode(len(*essKLV))
				// write the data and update the file position

				essBytes := make([]byte, len(pipe.info.key))
				copy(essBytes, pipe.info.key)
				essBytes = append(essBytes, berLength...)
				essBytes = append(essBytes, *essKLV...)

				_, err := w.Write(essBytes)
				if err != nil {
					return nil, fmt.Errorf("error encoding essence %v", err)
				}

				filePosition.totalByteCount += len(berLength) + len(*essKLV) + len(pipe.info.key)
			}

		}

	}

	// generate the unclocked metadata streams all at the end
	for i, dataStream := range unClockDataStreams {

		essPacket, essChanOpen := <-dataStream.pack.Packets

		if !essChanOpen {
			continue // @TODO check on the intended behaviour here
			// we may want to return an error instead
		}
		// upate the stream id for each generic parition
		filePosition.sID++

		err := writePartition(w, filePosition, headerName(genericStream, false, false), 0, []byte{}, essSetup.containerKeys)
		if err != nil {
			return nil, err
		}

		essKLV := essPacket.Data
		berLength := mxf2go.BEREncode(len(*essKLV))

		essBytes := make([]byte, len(dataStream.info.key))
		copy(essBytes, dataStream.info.key)
		essBytes = append(essBytes, berLength...)
		essBytes = append(essBytes, *essKLV...)

		_, err = w.Write(essBytes)
		if err != nil {
			return nil, fmt.Errorf("error encoding essence %v", err)
		}

		filePosition.totalByteCount += len(berLength) + len(*essKLV) + len(dataStream.info.key)
		// update the manifest options
		man := essPacket.MetaData
		manifesters[clockCount+i].Essence = append(manifesters[clockCount+i].Essence, *man)
		manifesters[clockCount+i].Common = dataStream.pack.OverViewData

	}

	// collect any errors from the data stream
	err := errs.Wait()
	if err != nil {
		return nil, err
	}

	partitionManifest = append(partitionManifest, manifesters...)

	filePosition.sID++ // update the SID for the manifest

	return partitionManifest, nil
}

var runin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 2, 4, 0}
var bodyin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 3, 4, 0}
var footerin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 4, 4, 0}
var genericStreamin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 3, 0x11, 0}

// update the 13th byte as a flag to show it binary or text

// 60e2b34.02050101.0d010201.01031100

const (
	header = iota
	body
	footer
	genericStream
)

// header name generates the partition pack and open/complete status of the oartition
func headerName(partition int, closed, complete bool) [16]byte {

	var name [16]byte
	// [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 3, 4,  0}
	// get the type
	switch partition {
	case header:
		name = runin
	case body:
		name = bodyin
	case footer:
		name = footerin
	case genericStream:
		// generic stream has a set layout, updating the closed complete changes its meaning
		return genericStreamin

	}

	// ammend to be refelct the pen and complete status

	switch {
	case closed && complete:
		name[14] = 04
	case !closed && complete:
		name[14] = 03
	case closed && !complete:
		name[14] = 02
	default /*closed && !complete*/ :
		name[14] = 01
	}

	/*open and incomplete 01h
	closed and incomplete 02h
	open and comepltee 03
	closed and complete 04*/
	return name
}

/*
var binaryFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00}

// var textFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x02, 0x00, 0x00}
var textFrameDesc = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x04, 0x01, 0x01, 0x05, 0x0e, 0x09, 0x06, 0x07, 0x01, 0x01, 0x01, 0x03}
var textFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x01, 0x01, 0x01}
var binaryClipKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x03, 0x00, 0x00}
var textClipKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x04, 0x00, 0x00}
*/

var manifestKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00}

func (mw *MrxWriter) uMIDFinish(esssenceCount int) {

	switch {
	case esssenceCount == 1: // len(mw.essenceList) == 1 {
		mw.writeInformation.mrxUMID.SMPTELabel[10] = 0xb
	case esssenceCount == 0: // len(mw.essenceList) ==  0 {
		mw.writeInformation.mrxUMID.SMPTELabel[10] = 0xf
	default:
		mw.writeInformation.mrxUMID.SMPTELabel[10] = 0xc
	}
	// mix is 0d
	// empty is of

	// update the time in two formats as well
	gTime := time.Now()
	Date := mxf2go.TDateStruct{Year: int16(gTime.Year()), Month: uint8(gTime.Month()), Day: uint8(gTime.Day())}
	Time := mxf2go.TTimeStruct{Hour: uint8(gTime.Hour()), Minute: uint8(gTime.Minute()), Second: uint8(gTime.Second())}

	mw.writeInformation.buildTimeTime = gTime
	mw.writeInformation.buildTime = mxf2go.TTimeStamp{Date: Date, Time: Time}
}

// RIP Layout is the simple layout of a partition in an mrx file
type RIPLayout struct {
	SID               uint32
	partitionPosition uint64
}

// RIPKey is the Byte key for the random index pack of the mrx file
var RIPKey = []byte{0x6, 0xe, 0x2B, 0x34, 0x02, 0x05, 0x01, 0x01, 0x0d, 0x01, 0x02, 0x01, 0x01, 0x11, 0x01, 0x00}

func rIPPack(partitions []RIPLayout) []byte {

	var ribBuffer bytes.Buffer
	// Write key then length
	ribBuffer.Write(RIPKey)
	// we can predict the length as it is an array so don't need to calculate after the pack is written
	berBytes := mxf2go.BEREncode(12*len(partitions) + 4)
	ribBuffer.Write(berBytes)

	// generate the SID and the position
	for _, part := range partitions {
		sid := make([]byte, 4)
		order.PutUint32(sid, part.SID)
		partOffset := make([]byte, 8)
		order.PutUint64(partOffset, part.partitionPosition)

		ribBuffer.Write(sid)
		ribBuffer.Write(partOffset)
	}

	// calculate the total length and append it
	totalLength := make([]byte, 4)
	order.PutUint32(totalLength, uint32(12*len(partitions)+16+len(berBytes)+4))
	ribBuffer.Write(totalLength)

	return ribBuffer.Bytes()
}

// header minimum required - material package and a source package
func (mw *MrxWriter) metaData(stream mrxLayout) []byte {

	essenceKeys := stream.containerKeys
	// tauidKeys is the genereated form of the essence keys 060e34...
	// and is used as part of the preface package
	tauidKeys := make([]mxf2go.TAUID, len(essenceKeys))
	for i, ek := range essenceKeys {
		var array8 [8]uint8

		copy(array8[:], ek[8:])

		tauidKeys[i] = mxf2go.TAUID{
			Data1: order.Uint32(ek[0:4]),
			Data2: order.Uint16(ek[4:6]),
			Data3: order.Uint16(ek[6:8]),
			Data4: mxf2go.TUInt8Array8(array8),
		}
	}

	// dynamic tags for the primer pack
	// the tag decrements down from 0xffff to 0x8000 if they do not have a predeclared value
	// tags is a map of dynamic and preallocated bytes and their long name
	primer := mxf2go.NewPrimer()

	// gtime is when the thing was written
	gTime := time.Now()
	Date := mxf2go.TDateStruct{Year: int16(gTime.Year()), Month: uint8(gTime.Month()), Day: uint8(gTime.Day())}
	Time := mxf2go.TTimeStruct{Hour: uint8(gTime.Hour()), Minute: uint8(gTime.Minute()), Second: uint8(gTime.Second())}

	mw.frameInformation.ContainerKeys = essenceKeys
	contentBytes, contentID := mw.contentStorage(primer, stream)

	idb, idid := identification(primer)
	//	isxdBytes := isxdHeader(tag, tags)

	// data essence track
	dataEss := mxf2go.TAUID{
		Data1: 101591860,
		Data2: 1025,
		Data3: 257,
		Data4: mxf2go.TUInt8Array8{01, 03, 02, 02, 03, 00, 00, 00}} //  060e2b34.04010101.01030202.03000000
	// descriptive essence track
	// @TODO check with 2057 that this is the corect key
	descTrack := mxf2go.TAUID{
		Data1: 101591860,
		Data2: 1025,
		Data3: 257,
		Data4: mxf2go.TUInt8Array8{01, 03, 02, 01, 0x10, 00, 00, 00}}
	GotData := mxf2go.TAUIDSet{}

	gotAll := false
	for _, s := range stream.dataStreams {
		switch {
		case s.clocked && !slices.Contains(GotData, dataEss):
			GotData = append(GotData, dataEss)
		case !slices.Contains(GotData, descTrack):
			GotData = append(GotData, descTrack)
		case len(GotData) == 2:
			gotAll = true
		}

		if gotAll {
			break
		}
	}

	// @TODO move to primer to seperate function
	pre := mxf2go.GPrefaceStruct{FormatVersion: mxf2go.TVersionType{VersionMajor: 1, VersionMinor: 3}, DescriptiveSchemes: GotData,
		ContentStorageObject: mxf2go.TStrongReference(contentID[:]), EssenceContainers: tauidKeys, InstanceID: mxf2go.TUUID(uuid.New()),
		FileLastModified: mxf2go.TTimeStamp{Date: Date, Time: Time}, IdentificationList: mxf2go.TIdentificationStrongReferenceVector{idid[:]},
		OperationalPattern: mxf2go.TAUID{
			Data1: 0x060e2b34,
			Data2: 0x0401,
			Data3: 0x0101,
			Data4: [8]byte{0x0d, 01, 02, 01, 01, 01, 01, 00},
		}}

	prefaceBytes, _ := pre.Encode(primer)

	Primer := primerEncode(primer)

	// add the preface
	Primer = append(Primer, prefaceBytes...)

	// add the content bytes
	Primer = append(Primer, contentBytes...)
	Primer = append(Primer, idb...)

	// generate the isxd header
	// Primer = append(Primer, isxdBytes...)
	return Primer // append(Primer, cb...)
}

func primerEncode(primer *mxf2go.Primer) []byte {
	Primer := []byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01, 0x0d, 0x01, 0x02, 0x01, 0x01, 0x05, 0x01, 0x00}
	length := []byte{0x83}

	tags := primer.Tags

	byte3 := order.AppendUint32([]byte{}, 8+uint32(len(tags))*18)
	//	fmt.Println(order.AppendUint32([]byte{}, 8+uint32(len(tags))*18), tags)
	length = append(length, byte3[1:]...) // has to be four byte long BER

	//	fmt.Println(length, "length")
	length = order.AppendUint32(length, uint32(len(tags)))
	length = order.AppendUint32(length, 18)
	// add the shorthnad nad long tags
	for full, shortHand := range tags {

		length = append(length, shortHand...)
		length = append(length, []byte(full)...)
	}

	Primer = append(Primer, length...)

	return Primer
}

// partition pack is the hardcoded structure
type partitionPack struct {
	Signature         [16]byte // Must be, hex: 06 0E 2B 34
	PartitionLength   int      // All but first block size
	MajorVersion      uint16   // Must be, hex: 01 00
	MinorVersion      uint16
	SizeKAG           uint32
	ThisPartition     uint64
	PreviousPartition uint64
	FooterPartition   uint64 // First block size
	HeaderByteCount   uint64
	IndexByteCount    uint64
	IndexSID          uint32
	BodyOffset        uint64
	BodySID           uint32
}

func encodePartition(header partitionPack, essenceKeys [][]byte) ([]byte, int) {

	var headerBytes bytes.Buffer
	headerBytes.Write(header.Signature[:]) // convert the array to a slice
	headerBytes.Write([]byte{0x83, 0, 0, byte(header.PartitionLength)})
	headerBytes.Write(order.AppendUint16([]byte{}, header.MajorVersion))
	headerBytes.Write(order.AppendUint16([]byte{}, header.MinorVersion))
	headerBytes.Write(order.AppendUint32([]byte{}, header.SizeKAG))
	headerBytes.Write(order.AppendUint64([]byte{}, header.ThisPartition))
	headerBytes.Write(order.AppendUint64([]byte{}, header.PreviousPartition))
	headerBytes.Write(order.AppendUint64([]byte{}, header.FooterPartition))
	headerBytes.Write(order.AppendUint64([]byte{}, header.HeaderByteCount))
	headerBytes.Write(order.AppendUint64([]byte{}, header.IndexByteCount))
	headerBytes.Write(order.AppendUint32([]byte{}, header.IndexSID))
	headerBytes.Write(order.AppendUint64([]byte{}, header.BodyOffset))
	headerBytes.Write(order.AppendUint32([]byte{}, header.BodySID))
	// extra bits which I haven't changed
	// 060E2B3404010101.0D01020101010100
	// get operational pattern
	op := generateOperationalPattern()
	headerBytes.Write(op[:]) // Operational pattern of the mrx file
	headerBytes.Write(order.AppendUint32([]byte{}, uint32(len(essenceKeys))))
	headerBytes.Write([]byte{0, 0, 0, 16})

	for _, essence := range essenceKeys {
		//	fmt.Println(string(essence))
		headerBytes.Write(essence)
	}

	return headerBytes.Bytes(), headerBytes.Len()
}

const (
	mrxTool = "Mr MXF's MRX golang command line tool"
)

// encode manifest generates the json bytes of a mainfest.
// using any previous manifests if required
func (mw *MrxWriter) encodeRoundTrip(setup *manifest.RoundTrip, manifesters []manifest.Overview, mrxChans mrxLayout, manifestCount int) ([]byte, error) {
	prevManifest := setup.Manifest
	prevManifestTag := manifest.TaggedManifest{Manifest: prevManifest}

	UUIDb, _ := mw.writeInformation.mrxUMID.MarshalText()
	destManifest := manifest.Manifest{UMID: string(UUIDb), MRXTool: mrxTool, Version: " 0.0.0.1"}

	// if it a manifest has been found
	if !reflect.DeepEqual(prevManifestTag.Manifest, manifest.Manifest{}) {
		history := prevManifest.History
		prevManifest.History = nil

		destManifest.History = append([]manifest.TaggedManifest{prevManifestTag}, history...)
	}
	// else continue as normal as there is no mainpulation of th eprevious manifest

	destManifest.DataStreams = manifesters
	// handle how many previous manifests are included in the manifest
	x := manifestCount

	switch {
	case x == -1 || x > len(destManifest.History):
		// do nothing, as the user has asked for all the history
	case x == 0: // if  0 do not assign
		destManifest.History = nil
	default: // else trim to the desired length
		destManifest.History = destManifest.History[:x]
	}

	// update the set up to contain the mainfest information
	setup.Manifest = destManifest

	if mrxChans.reorder {

		reorder := manifest.Configuration{Version: setup.Config.Version, Default: setup.Config.Default,
			StreamProperties: make(map[int]manifest.StreamProperties)}
		fwCount := 0
		clipWrapped := []int{}
		for i, mrxChan := range mrxChans.dataStreams {
			if mrxChan.clocked {
				// update the clocked position with this current one
				reorder.StreamProperties[fwCount] = setup.Config.StreamProperties[i]
				fwCount++
			} else {
				clipWrapped = append(clipWrapped, i)
			}
		}

		for _, i := range clipWrapped {
			reorder.StreamProperties[fwCount] = setup.Config.StreamProperties[i]
			fwCount++
		}

		setup.Config = reorder
	}

	manb, err := json.MarshalIndent(setup, "", "    ")

	if err != nil {
		return nil, fmt.Errorf("error encoding the manifest: %v", err)
	}

	length := mxf2go.BEREncode(len(manb))

	var buffer bytes.Buffer
	buffer.Write(manifestKey[:])
	buffer.Write(length)
	buffer.Write(manb)

	return buffer.Bytes(), nil
}
