// package encode is for encoding mrx files.
// It contains a generic interface for you to include your own data inputs for mxf files
package encode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gitlab.com/mm-eng/generatedmrx"
	"golang.org/x/sync/errgroup"
)

// The Writer interface is a way to plug in the essence generator into an MRX file to save
// the essence, in an generic way without having to deal with the MRX file internal data, such as headers.
type Writer interface {

	// GetEssenceKeys gives the array of the essence Keys
	// to be used in this mrx file
	GetStreamInformation() (StreamInformation, error)

	// The essence Pipe returns streams of KLV and their associated metadata
	// Each channel represents a seperate stream. And will be seperated by a partition and stream ID
	EssenceChannels(chan *ChannelPackets) error

	// RoundTrip gets the json in the target location
	// this will be different for streams etc
	GetRoundTrip() (*Roundtrip, error)
}

// ChannelPackets contains the user metadata for a partition
// and the channel for streaming data.
type ChannelPackets struct {
	OverViewData GroupProperties
	Packets      chan *DataCarriage
}

// DataCarriage contains the essence
// and any metadata generated during it's construction.
type DataCarriage struct {
	Data     *[]byte
	MetaData *EssenceProperties
}

type StreamInformation struct {
	// ChannelCount is the number of channels
	ChannelCount int
	// Essence Keys are the essence keys of each data type
	// in the order they are to be streamed to the encoder
	EssenceKeys []EssenceKey
}

// MrxEncodeOptions are the enocding parameters. ManifestHistoryCount is
// the number of previous manifest files to be inlcuded (if possible)
type MrxEncodeOptions struct {
	ManifestHistoryCount int
	ConfigOverWrite      []byte
}

// Write writes the data to an mrx file, default options are used if MrxEncodeOptions is nil
func (mw *MxfWriter) Write(w io.Writer, encodeOptions *MrxEncodeOptions) error {

	// get the mrxWriter methods
	mrxwriter := mw.saver

	if mrxwriter == nil {
		return fmt.Errorf("Error saving, no essence extraction methods available")
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

	//this is where the config update would come in

	// merge the user options and the parsed information
	cleanStream := streamClean(essenceStream, round.Config)

	// get the essence keys
	containerKeys := cleanStream.containerKeys
	// set the writer infromation
	// @TODO check its the clean stream stuff
	mw.frameInformation.StreamTimeLine = round.Config

	// generate the UMDID for this mrx file
	mw.uMIDFinish(len(containerKeys))

	// metadata set up
	headerMeta := mw.metaData(cleanStream)

	// have an object to reference how far through the file generation we are as we generate it
	filePosition := &partitionPosition{partitions: []RIPLayout{}, totalByteCount: 0, prevPartition: 0}

	// write the header partition
	writePartition(w, filePosition, headerName(header, false, false), 0, headerMeta, containerKeys)

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
	//write the manifest and update the position
	writePartition(w, filePosition, headerName(body, false, false), 0, []byte{}, containerKeys)
	w.Write(manifestBytes)
	filePosition.totalByteCount += len(manifestBytes)

	//check or essence extraction error handling
	//set the SID back to  0 at the end, then write the footer
	filePosition.sID = 0
	writePartition(w, filePosition, headerName(footer, true, true), uint64(filePosition.totalByteCount), headerMeta, containerKeys)

	// Finally the RIP
	w.Write(rIPPack(filePosition.partitions))

	return nil
}

// essence filter contains the properties of an stream
type channelProperties struct {
	clocked         bool
	key             []byte
	frameRate       generatedmrx.TRational
	frameMultiplier int
	nameSpace       string
}

type mrxLayout struct {
	dataStreams   []channelProperties
	containerKeys [][]byte
	baseFrameRate generatedmrx.TRational
	//
	isxdflag bool
	// reorder flags is framewrapped data is declared after clip wrapped
	// so that the config can be reorderd when it is saved as part of the mxf file
	// for roundtripping
	reorder bool
}

// stream clean goes through the esesnce
// updating to match the infomration the user provided and sotring out the container
func streamClean(foundStream StreamInformation, userStream Configuration) mrxLayout {

	var fullStream mrxLayout
	var base bool
	var clip bool

	cleanEssence := make([]channelProperties, foundStream.ChannelCount)

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
			fmt.Sscanf(getFrame, "%v/%v", &num, &dom)

			if num == 0 || dom == 0 {
				// @TODO implement a better way to handle thos
				num, dom = 24, 1
				fullStream.baseFrameRate = generatedmrx.TRational{Numerator: 24, Denominator: 1}
			}

			if !base {

				fullStream.baseFrameRate = generatedmrx.TRational{Numerator: num, Denominator: dom}
				base = true
				// only one item in this content package as it is the frame rate essence
				cleanEssence[i].frameMultiplier = 1
				//	essenceKey[13] = 1
			} else {

				// @TODO include some error handling for when its greater than 127
				// or less than 1
				cleanEssence[i].frameMultiplier = int(num) / int(fullStream.baseFrameRate.Numerator)
				//essenceKey[13] = byte(int(num) / int(fullStream.baseFrameRate.Numerator))
			}
			essenceKey[13] = byte(cleanEssence[i].frameMultiplier)
			cleanEssence[i].frameRate = generatedmrx.TRational{Numerator: num, Denominator: dom}

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

	return fullStream
}

type partitionPosition struct {
	partitions     []RIPLayout
	totalByteCount int
	prevPartition  int
	sID            int
}

func writePartition(w io.Writer, filePosition *partitionPosition, header [16]byte, footerPos uint64, headerMeta []byte, essenceKeys [][]byte) {
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
	w.Write(partitionBytes)
	w.Write(headerMeta)
}

func encodeEssence(w io.Writer, filePosition *partitionPosition, mrxwriter Writer, essSetup mrxLayout) ([]Overview, error) {

	// set up the partition channels, generating as many channels as there are streams
	essenceContainers := make(chan *ChannelPackets, len(essSetup.dataStreams))

	clockCount := 0

	// get the clcoked essence positions

	for _, baseKey := range essSetup.dataStreams {

		if baseKey.clocked {

			clockCount++
		}

	}

	//essenceKeys := essSetup.EssenceKeys
	// use errs to handle errors while running concurrently
	// this is to allow us to use the channels
	errs, _ := errgroup.WithContext(context.Background())

	//initiate the klv stream
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

	//set up the datastreams from the input
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
	manifesters := make([]Overview, len(essSetup.dataStreams))
	filePosition.sID = 1
	partitionManifest := []Overview{}

	//set up a stream flag
	availableEssence := true

	if len(clockDataStreams) == 0 {
		availableEssence = false
	}

	// multiplex the framewrapped data together
	if availableEssence {
		writePartition(w, filePosition, headerName(body, false, false), 0, []byte{}, essSetup.containerKeys)
	}

	for availableEssence {

		for i, pipe := range clockDataStreams {

			// to stop extra data being written after the first channel is closed
			for j := 0; j < pipe.info.frameMultiplier; j++ {

				essPacket, essChanOpen := <-pipe.pack.Packets

				if !essChanOpen && i == 0 {

					//only stop the stream when the key data has finished being sent
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
				berLength := generatedmrx.BEREncode(len(*essKLV))
				// write the data and update the file position
				w.Write(pipe.info.key)
				w.Write(berLength) // calculate the length of the data
				w.Write(*essKLV)

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
		//upate the stream id for each generic parition
		filePosition.sID++

		writePartition(w, filePosition, headerName(genericStream, false, false), 0, []byte{}, essSetup.containerKeys)

		essKLV := essPacket.Data
		berLength := generatedmrx.BEREncode(len(*essKLV))
		w.Write(dataStream.info.key)
		w.Write(berLength)
		w.Write(*essKLV)

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

	for _, mani := range manifesters {
		partitionManifest = append(partitionManifest, mani)
	}
	filePosition.sID++ //update the SID for the manifest

	return partitionManifest, nil
}

var runin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 2, 4, 0}
var bodyin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 3, 4, 0}
var footerin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 4, 4, 0}
var genericStreamin = [16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 3, 0x11, 0}

// update the 13th byte as a flag to show it binary or text

//60e2b34.02050101.0d010201.01031100

const (
	header = iota
	body
	footer
	genericStream
)

// header name generates the partition pack and open/complete status of the oartition
func headerName(partition int, closed, complete bool) [16]byte {

	var name [16]byte
	//[16]byte{6, 14, 43, 52, 2, 5, 1, 1, 13, 1, 2, 1, 1, 3, 4,  0}
	//get the type
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

	if closed && complete {
		name[14] = 04
	} else if !closed && complete {
		name[14] = 03
	} else if closed && !complete {
		name[14] = 02
	} else /*closed && !complete*/ {
		name[14] = 01
	}

	/*open and incomplete 01h
	closed and incomplete 02h
	open and comepltee 03
	closed and complete 04*/
	return name
}

var binaryFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00}

// var textFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x02, 0x00, 0x00}
var textFrameDesc = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x04, 0x01, 0x01, 0x05, 0x0e, 0x09, 0x06, 0x07, 0x01, 0x01, 0x01, 0x03}
var textFrameKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x01, 0x01, 0x01}
var binaryClipKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x03, 0x00, 0x00}
var textClipKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x04, 0x00, 0x00}

var manifestKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00}

func (mw *MxfWriter) uMIDFinish(esssenceCount int) {

	if esssenceCount == 1 { //len(mw.essenceList) == 1 {
		mw.writeInformation.mrxUMID.SMPTELabel[10] = 0xb
	} else if esssenceCount == 0 { //len(mw.essenceList) ==  0 {
		mw.writeInformation.mrxUMID.SMPTELabel[10] = 0xf
	} else {
		mw.writeInformation.mrxUMID.SMPTELabel[10] = 0xc
	}
	// mix is 0d
	// empty is of

	//update the time in two formats as well
	gTime := time.Now()
	Date := generatedmrx.TDateStruct{Year: int16(gTime.Year()), Month: uint8(gTime.Month()), Day: uint8(gTime.Day())}
	Time := generatedmrx.TTimeStruct{Hour: uint8(gTime.Hour()), Minute: uint8(gTime.Minute()), Second: uint8(gTime.Second())}

	mw.writeInformation.buildTimeTime = gTime
	mw.writeInformation.buildTime = generatedmrx.TTimeStamp{Date: Date, Time: Time}
}

// RIP Layout is the simple layout of a partition in an mrx file
type RIPLayout struct {
	SID               uint32
	partitionPosition uint64
}

var RIPKey = []byte{0x6, 0xe, 0x2B, 0x34, 0x02, 0x05, 0x01, 0x01, 0x0d, 0x01, 0x02, 0x01, 0x01, 0x11, 0x01, 0x00}

func rIPPack(partitions []RIPLayout) []byte {

	var ribBuffer bytes.Buffer
	// Write key then length
	ribBuffer.Write(RIPKey)
	// we can predict the length as it is an array so don't need to calculate after the pack is written
	berBytes := generatedmrx.BEREncode(12*len(partitions) + 4)
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

	//calculate the total length and append it
	totalLength := make([]byte, 4)
	order.PutUint32(totalLength, uint32(12*len(partitions)+16+len(berBytes)+4))
	ribBuffer.Write(totalLength)

	return ribBuffer.Bytes()
}

// header minimum required - material package and a source package
func (mw *MxfWriter) metaData(stream mrxLayout) []byte {

	essenceKeys := stream.containerKeys
	// tauidKeys is the genereated form of the essence keys 060e34...
	// and is used as part of the preface package
	tauidKeys := make([]generatedmrx.TAUID, len(essenceKeys))
	for i, ek := range essenceKeys {
		var array8 [8]uint8

		for j, arr := range ek[8:] {
			array8[j] = arr
		}

		tauidKeys[i] = generatedmrx.TAUID{
			Data1: order.Uint32(ek[0:4]),
			Data2: order.Uint16(ek[4:6]),
			Data3: order.Uint16(ek[6:8]),
			Data4: generatedmrx.TUInt8Array8(array8),
		}
	}

	// dynamic tags for the primer pack
	// the tag decrements down from 0xffff to 0x8000 if they do not have a predeclared value
	// tags is a map of dynamic and preallocated bytes and their long name
	tagStart := uint16(0xfffe)
	tag := &tagStart
	tags := make(map[string][]byte)

	// gtime is when the thing was written
	gTime := time.Now()
	Date := generatedmrx.TDateStruct{Year: int16(gTime.Year()), Month: uint8(gTime.Month()), Day: uint8(gTime.Day())}
	Time := generatedmrx.TTimeStruct{Hour: uint8(gTime.Hour()), Minute: uint8(gTime.Minute()), Second: uint8(gTime.Second())}

	mw.frameInformation.ContainerKeys = essenceKeys
	contentBytes, contentID := mw.contentStorage(tag, tags, stream)

	idb, idid := identification(tag, tags)
	//	isxdBytes := isxdHeader(tag, tags)

	// @TODO move to primer to seperate function
	pre := generatedmrx.GPrefaceStruct{FormatVersion: generatedmrx.TVersionType{VersionMajor: 1, VersionMinor: 3}, DescriptiveSchemes: generatedmrx.TAUIDSet{productID},
		ContentStorageObject: generatedmrx.TStrongReference(contentID[:]), EssenceContainers: tauidKeys, InstanceID: generatedmrx.TUUID(uuid.New()),
		FileLastModified: generatedmrx.TTimeStamp{Date: Date, Time: Time}, IdentificationList: generatedmrx.TIdentificationStrongReferenceVector{idid[:]},
		OperationalPattern: generatedmrx.TAUID{
			Data1: 0x060e2b34,
			Data2: 0x0401,
			Data3: 0x0101,
			Data4: [8]byte{0x0d, 01, 02, 01, 01, 01, 01, 00},
		}}

	prefaceBytes, _ := pre.Encode(tag, tags)

	Primer := primerEncode(tags)

	// add the preface
	Primer = append(Primer, prefaceBytes...)

	//add the content bytes
	Primer = append(Primer, contentBytes...)
	Primer = append(Primer, idb...)

	// generate the isxd header
	//Primer = append(Primer, isxdBytes...)
	return Primer //append(Primer, cb...)
}

func primerEncode(tags map[string][]byte) []byte {
	Primer := []byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01, 0x0d, 0x01, 0x02, 0x01, 0x01, 0x05, 0x01, 0x00}
	length := []byte{0x83}
	byte3 := order.AppendUint32([]byte{}, 8+uint32(len(tags))*18)
	//	fmt.Println(order.AppendUint32([]byte{}, 8+uint32(len(tags))*18), tags)
	length = append(length, byte3[1:]...) // has to be four byte long BER

	//	fmt.Println(length, "length")
	length = order.AppendUint32(length, uint32(len(tags)))
	length = order.AppendUint32(length, 18)
	// add the shorthnad nad long tags
	for key, full := range tags {

		length = append(length, []byte(key)...)
		length = append(length, full...)
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
	headerBytes.Write(header.Signature[:]) //convert the array to a slice
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
	//extra bits which I haven't changed
	//060E2B3404010101.0D01020101010100
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
func (mw *MxfWriter) encodeRoundTrip(setup *Roundtrip, manifesters []Overview, mrxChans mrxLayout, manifestCount int) ([]byte, error) {
	prevManifest := setup.Manifest

	prevManifestTag := TaggedManifest{Manifest: prevManifest}

	UUIDb, _ := mw.writeInformation.mrxUMID.MarshalText()
	manifest := Manifest{UMID: string(UUIDb), MRXTool: mrxTool, Version: " 0. 0. 0.1"}

	//if it a manifest has been found
	if !reflect.DeepEqual(prevManifestTag.Manifest, Manifest{}) {
		history := prevManifest.History
		prevManifest.History = nil

		manifest.History = append([]TaggedManifest{*&prevManifestTag}, history...)
	}
	// else continue as normal as there is no mainpulation of th eprevious manifest

	manifest.DataStreams = manifesters
	//handle how many previous manifests are included in the manifest
	x := manifestCount
	if x == -1 || x > len(manifest.History) {
		// do nothing, as the user has asked for all the history
	} else if x == 0 { // if  0 do not assign
		manifest.History = nil
	} else { //else trim to the desired length
		manifest.History = manifest.History[:x]
	}

	// update the set up to contain the mainfest information
	setup.Manifest = manifest

	// STREAM ID reoredering
	/*

		push the streamset up through
		if reorder = true




	*/

	if mrxChans.reorder {

		reorder := Configuration{Version: setup.Config.Version, Default: setup.Config.Default,
			StreamProperties: make(map[int]streamProperties)}
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

	length := generatedmrx.BEREncode(len(manb))

	var buffer bytes.Buffer
	buffer.Write(manifestKey[:])
	buffer.Write(length)
	buffer.Write(manb)

	return buffer.Bytes(), nil
}
