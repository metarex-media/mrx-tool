// Package decode handles the MRX decoding
package decode

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/metarex-media/mrx-tool/klv"

	mxf2go "github.com/metarex-media/mxf-to-go"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// MRXStructureExtractor takes an MRX stream and decodes the layout to the writer.
func MRXStructureExtractor(mrxStream io.Reader, w io.Writer, contentPackageLimit []int, jsonFile bool) error {

	internalLayout, err := klvStream(mrxStream, contentPackageLimit, 10)
	// fmt.Println(internalLayout, err)
	if err != nil {
		return err
	}

	var layoutBytes []byte

	if jsonFile {
		layoutBytes, err = json.MarshalIndent(internalLayout, "", "    ")

	} else {
		layoutBytes, err = yaml.Marshal(internalLayout)
	}

	if err != nil {
		return err
	}

	// write the yaml
	_, err = w.Write(layoutBytes)

	return err
}

// Dataformat has the stream ID and the data within
type DataFormat struct {
	MRXID     string
	FrameRate string
	Data      [][]byte
}

// Extract streamData takes an MRX file and
// extracts each metadata stream into a seperate data stream
// in the order it is found in the file.
func ExtractStreamData(mrxStream io.Reader) ([]*DataFormat, error) {

	// get the decoder here
	// utilise essenceExtract copy and pasting code to do something with it.
	// then work on making something more generic
	klvChan := make(chan *klv.KLV, 1000)

	return essenceExtract(mrxStream, klvChan)
}

func klvStream(stream io.Reader, contentPackageLimit []int, size int) (essenceLayout, error) {

	klvChan := make(chan *klv.KLV, 100)

	decoder, err := MRXReader(stream, klvChan, size)

	if err != nil {
		return essenceLayout{}, err
	}

	// extract the limited packages
	containers := contentPackageLimiter(decoder.allKeys, decoder.containers, contentPackageLimit)

	streamEssence := essenceLayout{}
	streamEssence.Partitions = containers

	return streamEssence, nil

}

type essenceLayout struct {

	// warnings - soft errors like thigns being in the header
	// {warning: "essence found in header partition", essenceLocation : 0
	// warning : No Random Index Pack Found

	// timings 1 minute frame 24 etc
	// Partitions is the list of essence containing paritions in the order
	// they were found in the mrx file
	Warnings   []warning   `yaml:"Warnings,omitempty" json:"Warnings,omitempty"`
	Partitions []container `yaml:"Partitions" json:"Partitions"`
}

// MRXReader reads an MRX stream, then buffers through the klv channel breaking down the contents
// into a go struct.
func MRXReader(stream io.Reader, buffer chan *klv.KLV, size int) (*mrxDecoder, error) { // wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	// use errs to handle errors while running concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream
	errs.Go(func() error {
		return klv.StartKLVStream(stream, buffer, size)

	})

	countStart := 0
	md := &mrxDecoder{Primer: make(map[string]string), Unknown: make(map[string]mxf2go.EssenceInformation), unknownCount: &countStart}

	// initiate the klv handling stream
	errs.Go(func() error {

		// @TODO: stop the klv channel blocking if this go function returns early before
		// reading everything.
		// currently i empty the channel at the end to run everything.
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

				if klvItem.Key[13] == 17 {
					// return the end of the file
					// we don't need to read the RIP

					return nil
				} else {
					// decode the partition
					err := md.partitionDecode(klvItem, buffer)

					if err != nil {
						return err
					}
				}
			} else {
				// decode as essence
				err := md.essenceDecode(klvItem)
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
	return md, nil
}

func fullName(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15])
}

// partition name generates the name string removing the variable bits
func partitionName(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	// "060e2b34.020501  .0d010201.0103  00"
	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x  .%02x%02x%02x%02x.%02x    %02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[15])
}

func contentKey(packages []contentPackage) string { // return the key of the first pacakge to check for patterns

	contents := packages[len(packages)-1]

	if len(contents.ContentPackage) == 0 {
		return "faker"
	} else {
		return contents.ContentPackage[0].Key
	}

}

func (md *mrxDecoder) partitionDecode(klvItem *klv.KLV, metadata chan *klv.KLV) error {

	if md.partitionCount != 0 {
		if int(md.average.count) != len(md.currentContainer.ContentPackages) {
			md.average.Update(float64(md.currentContainer.ContentPackages[len(md.currentContainer.ContentPackages)-1].ContentPackageLength))
		}

		// check if there's any contents
		if len(md.currentContainer.ContentPackages) > 0 {
			// check there's any keys in the conents
			if len(md.currentContainer.ContentPackages[0].ContentPackage) > 0 {
				// call the stats
				finalAvg := md.average.finalise()
				// use a pointer of the result if an average has been calculated
				if finalAvg.Mean != 0 {
					md.currentContainer.Stats = &finalAvg
				}
				md.currentContainer.ContentPackageCount = len(md.currentContainer.ContentPackages)

				//	md.currentContainer.ContentPackages = contents //essences

			}
		}

		if md.currentContainer.PartitionType == "header" && len(md.currentContainer.ContentPackages[0].ContentPackage) > 0 {
			md.currentContainer.Warning = &warning{Message: "Essence found in the partition header"}

		}

		// update the partition infomratino before resetting it to 0
		md.currentContainer.EssenceByteCount = md.byteCount

		if len(md.currentContainer.ContentPackages[0].ContentPackage) == 0 {
			md.currentContainer.ContentPackages = []contentPackage{}
		}

		md.containers = append(md.containers, md.currentContainer)
		md.allKeys = append(md.allKeys, md.currentContainer.ContentPackages...)

	}

	// generate a new partition
	md.currentContainer = container{}
	//	shift, lengthlength := klvItem
	partitionLayout := partitionExtract(klvItem)
	md.currentContainer.PartitionType, md.currentContainer.HeaderLength = partitionLayout.PartitionType, partitionLayout.TotalHeaderLength

	md.currentContainer.ContentPackages = []contentPackage{{ContentPackage: []keyLength{}}}
	md.average = stats{Minimum: math.MaxInt}

	// flush out the header metadata
	// as it is not used yet (apart from the primer)
	flushedMeta := 0
	for flushedMeta < int(partitionLayout.HeaderByteCount) {
		flush, open := <-metadata

		if !open {
			return fmt.Errorf("error when using klv data klv stream interrupted")
		}
		// fmt.Println(ok, partitionLayout.HeaderByteCount, flushedMeta, partitionLayout.PartitionType)
		// fmt.Println(flush.Key, ok, partitionLayout.HeaderByteCount, flushedMeta)
		flushedMeta += flush.TotalLength()

		if string(flush.Key) == string([]byte{6, 0xe, 0x2b, 0x34, 2, 5, 1, 1, 0xd, 01, 02, 01, 01, 05, 01, 00}) {
			primerUnpack(flush.Value, md.Primer)
		}
	}

	// add the index table if there are some
	if partitionLayout.IndexTable {
		//	index table is after all the metadata
		index := <-metadata
		filledtable, err := indexUnpack(index, md.Primer)
		if err != nil {

			return err
		}

		md.currentContainer.IndexTable = filledtable
		//	fmt.Println(md.currentContainer.IndexTable)
	}

	// move the partition along and increment the partition counts
	md.partitionCount++
	md.byteCount = 0

	// increase the length by the name etc
	md.globalPosition += md.currentContainer.HeaderLength

	// position += md.currentContainer.HeaderLength

	return nil
}

func (md *mrxDecoder) essenceDecode(klvItem *klv.KLV) error {

	if md.partitionCount == 0 {
		// invalid as essence is called before the partition
		return fmt.Errorf("Invalid MRX File")
	}

	name := fullName(klvItem.Key)
	klvTotal := klvItem.TotalLength()
	// partLength, BERlength := klv.BerDecode(partStream[position+16 : position+16+berDistance : position+16+berDistance])

	// skip klv fill items
	if name == "060e2b34.01010102.03010210.01000000" || name == "060e2b34.01010101.03010210.01000000" || name == "060e2b34.01020101.03010210.01000000" {
		// fmt.Println(BERlength + partLength + 16)
		md.byteCount += klvTotal
		md.globalPosition += klvTotal

		return nil
	}

	// see if the essence has a key that correlates to the registers
	gotType := ExtractEssenceType(klvItem.Key, md.Unknown, md.unknownCount)
	contentSymbol := gotType.Symbol
	desc := gotType.Definition

	// check if is a new content pack or not
	if name == contentKey(md.currentContainer.ContentPackages) {

		md.average.Update(float64(md.currentContainer.ContentPackages[len(md.currentContainer.ContentPackages)-1].ContentPackageLength))
		md.currentContainer.ContentPackages = append(md.currentContainer.ContentPackages, contentPackage{ContentPackage: []keyLength{{Key: name,
			Length: len(klvItem.Value), FileOffset: md.globalPosition,
			Symbol: contentSymbol, TotalByteCount: klvTotal, Description: desc}}})
	} else {
		md.currentContainer.ContentPackages[len(md.currentContainer.ContentPackages)-1].ContentPackage = append(md.currentContainer.ContentPackages[len(md.currentContainer.ContentPackages)-1].ContentPackage,
			keyLength{Key: name, Length: len(klvItem.Value), FileOffset: md.globalPosition,
				Symbol: contentSymbol, TotalByteCount: klvTotal, Description: desc})
	}

	// update the content length and place in the stream
	md.currentContainer.ContentPackages[len(md.currentContainer.ContentPackages)-1].ContentPackageLength += klvTotal

	md.globalPosition += klvTotal
	md.byteCount += klvTotal

	return nil

}

func indexUnpack(indexTable *klv.KLV, primer map[string]string) (map[string]any, error) {

	// fmt.Println(fullName(indexTable[0:16]))
	decodeStructure, _ := decodeBuilder(indexTable.Key[5])
	/*

		decodeMethod get what the sixth byte
		keylength


	*/
	index := make(map[string]any)
	key := 0
	decoders := mxf2go.GIndexTableSegment
	for key < len(indexTable.Value) {
		newKey, keyLength := decodeStructure.keyFunc(indexTable.Value[key : key+decodeStructure.keyLen : key+decodeStructure.keyLen])
		length, sizeLength := decodeStructure.lengthFunc(indexTable.Value[key+keyLength : key+keyLength+decodeStructure.keyLen : key+keyLength+decodeStructure.keyLen])

		fullUL, okUL := primer[newKey]
		target := "urn:smpte:ul:" + fullUL
		if !okUL { // search in the default areas if the primer is lacking
			target = mxf2go.ShortHandLookUp[newKey]
		}

		decodeMethod, ok := decoders[target]
		if ok {
			res, _ := decodeMethod.Decode(indexTable.Value[key+keyLength+sizeLength : key+keyLength+sizeLength+int(length)])
			index[decodeMethod.UL] = res
		}

		key += sizeLength + keyLength + int(length)
	}

	// remove the long indexArray for the moment
	delete(index, "IndexEntryArray")

	return index, nil
}

type mxfPartition struct {
	Signature         string // Must be, hex: 06 0E 2B 34
	PartitionLength   int    // All but first block size
	MajorVersion      uint16 // Must be, hex: 01 00
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

	// useful information from the partition
	PartitionType     string
	IndexTable        bool
	TotalHeaderLength int
	MetadataStart     int
}

var (
	order = binary.BigEndian
)

func partitionExtract(partionKLV *klv.KLV) mxfPartition {

	// error checking on the length is done before parsing the stream to this function

	var partPack mxfPartition
	switch partionKLV.Key[13] {
	case 02:
		// header
		partPack.PartitionType = "header"
	case 03:
		// body
		partPack.PartitionType = "body"
	case 04:
		// footer
		partPack.PartitionType = "footer"
	default:
		// is nothing
		partPack.PartitionType = "invalid"
		return partPack
	}

	partPack.Signature = fullName(partionKLV.Key)

	//	packLength, lengthlength := berDecode(ber)
	partPack.PartitionLength = partionKLV.LengthValue
	partPack.MajorVersion = order.Uint16(partionKLV.Value[:2:2])
	partPack.MinorVersion = order.Uint16(partionKLV.Value[2:4:4])
	partPack.SizeKAG = order.Uint32(partionKLV.Value[4:8:8])
	partPack.ThisPartition = order.Uint64(partionKLV.Value[8:16:16])
	partPack.PreviousPartition = order.Uint64(partionKLV.Value[16:24:24])
	partPack.FooterPartition = order.Uint64(partionKLV.Value[24:32:32])
	partPack.HeaderByteCount = order.Uint64(partionKLV.Value[32:40:40])
	partPack.IndexByteCount = order.Uint64(partionKLV.Value[40:48:48])
	partPack.IndexSID = order.Uint32(partionKLV.Value[48:52:52])
	partPack.BodyOffset = order.Uint64(partionKLV.Value[52:60:60])
	partPack.BodySID = order.Uint32(partionKLV.Value[60:64:64])

	kag := int(partPack.SizeKAG)
	headerLength := int(partPack.HeaderByteCount)
	indexLength := int(partPack.IndexByteCount)

	totalLength := kag + headerLength + indexLength
	partPack.MetadataStart = kag

	if kag == 1 {
		packLength := partionKLV.TotalLength()
		totalLength += packLength - kag
		partPack.MetadataStart = packLength
		// else metadata start is the kag
	}

	if indexLength > 0 {

		// develop and index table body to use
		partPack.IndexTable = true
	}

	partPack.TotalHeaderLength = totalLength

	// partion extract returns the type of partition and the length.
	//	fmt.Println(partPack, "pack here")
	//	fmt.Println(partPack, totalLength, "My partition oack")
	return partPack
}

type mrxDecoder struct {
	Primer       map[string]string
	Unknown      map[string]mxf2go.EssenceInformation
	unknownCount *int

	// internal layout of the mrx file
	containers []container
	allKeys    []contentPackage

	// internal positions measurements
	partitionCount, byteCount, globalPosition int

	// container holder for each partition
	currentContainer container
	average          stats
}

// errors: [{error: "essence found in header partition", location "header"}]
type container struct {
	PartitionType       string `yaml:"PartitionType" json:"PartitionType"`
	HeaderLength        int    `yaml:"HeaderLength" json:"HeaderLength"`
	EssenceByteCount    int    `yaml:"EssenceByteCount" json:"EssenceByteCount"`
	ContentPackageCount int    `yaml:"ContentPackageCount" json:"ContentPackageCount"`
	// Optional Extras that give more info about each partition, depending on its layout
	IndexTable      map[string]any   `yaml:"IndexTable,omitempty" json:"IndexTable,omitempty"`
	Warning         *warning         `yaml:"Warning,omitempty" json:"Warning,omitempty"`
	ContentPackages []contentPackage `yaml:"ContentPackages,omitempty" json:"ContentPackages,omitempty"`

	Stats *stats `yaml:"ContentPackageStatistics,omitempty" json:"ContentPackageStatistics,omitempty"`
}

type stats struct {
	Mean              float64 `yaml:"Mean" json:"Mean"`
	Variance          float64 `yaml:"Variance" json:"Variance"`
	StandardDeviation float64 `yaml:"StandardDeviation" json:"StandardDeviation"`
	count             float64
	m2                float64

	Minimum int `yaml:"Minimum" json:"Minimum"`
	Maxium  int `yaml:"Maximum" json:"Maximum"`
}

// using Welford's online algorithm
// en.wikipedia.org/wiki/Algorithms_for_calculating_variance
func (s *stats) Update(update float64) {
	s.count++

	delta := update - s.Mean
	s.Mean += (delta / s.count)
	// fmt.Println(s.Mean)
	delta2 := update - s.Mean
	s.m2 += delta * delta2

	if int(update) > s.Maxium {
		s.Maxium = int(update)
	}

	if int(update) < s.Minimum {
		s.Minimum = int(update)
	}

	// fmt.Println(s.m2, delta, delta2, s.Mean)

}

func (s *stats) finalise() stats {

	if s.count > 2 {
		s.Variance, s.StandardDeviation = s.m2/(s.count), math.Sqrt(s.m2/(s.count))
	} else {
		s.Mean, s.Variance, s.StandardDeviation = 0, 0, 0
		s.Maxium, s.Minimum = 0, 0

	}

	return *s
}

type warning struct {
	Message string `yaml:"Message"`
}

type contentPackage struct {
	ContentPackage       []keyLength `yaml:"ContentPackage,omitempty"`
	ContentPackageLength int         `yaml:"ContentPackageLength,omitempty"`
	// keep is not exported and is used for the internal filtering
	keep bool
}

// tag everything
// have omit empty so the index table can be the same form as standard essence
type keyLength struct {
	Key         string `yaml:"Key"`
	Symbol      string `yaml:"Symbol,omitempty"`
	Description string `yaml:"Description"`
	FileOffset  int    `yaml:"FileOffset"`
	Length      int    `yaml:"Length"`
	//	ElementLength  int
	TotalByteCount      int `yaml:"TotalByteCount"`
	TotalContainerCount int `yaml:"TotalContainerCount,omitempty"`

	// blue cheese or not model - this will give labels like sound etc
	//
}

const (
	prefix = "urn:smpte:ul:"
)

// ExtractEssenceType returns the essence information associated with a essence Key,
// if it a matching key found.
func ExtractEssenceType(ul []byte, matches map[string]mxf2go.EssenceInformation, pos *int) mxf2go.EssenceInformation {
	// prefix := "urn:smpte:ul:"

	if ess, ok := mxf2go.EssenceLookUp[prefix+fullNameTwo(ul)]; ok {
		return ess
	}

	if ess, ok := mxf2go.EssenceLookUp[prefix+fullNameOne(ul)]; ok {
		return ess
	}

	if ess, ok := mxf2go.EssenceLookUp[prefix+FullName(ul)]; ok {
		return ess
	}

	return unknownEssence(ul, matches, pos)
}

func unknownEssence(ul []byte, matches map[string]mxf2go.EssenceInformation, pos *int) mxf2go.EssenceInformation {

	if ess, ok := matches[string(ul)]; ok {
		return ess

	} else {
		sym := fmt.Sprintf("SystemItemTBD%v", *pos)

		newEss := mxf2go.EssenceInformation{Symbol: sym, UL: prefix + fullNameOne(ul)}
		matches[string(ul)] = newEss
		*pos++

		return newEss
	}

}

func fullNameTwo(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x7f%02x7f",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[14])
}

func fullNameOne(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x7f",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14])
}

func FullName(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15])
}
