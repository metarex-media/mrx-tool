package mrxUnitTest

import (
	"encoding/binary"
	"fmt"

	"github.com/metarex-media/mrx-tool/klv"

	. "github.com/onsi/gomega"
)

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

func (l *layout) partitionDecode(klvItem *klv.KLV, metadata chan *klv.KLV) error {
	// /	e.essenceCount = 0
	//	shift, lengthlength := klvItem

	partitionLayout := partitionExtract(klvItem)

	seg := newSegmentTest(l.testLog, fmt.Sprintf("Partiton %0d Tests", len(l.Rip))) // the length of the RIP gives the relative partition count
	tester := NewGomegaWithT(seg)

	seg.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
		return tester.Expect(uint64(l.TotalByteCount)).To(Equal(partitionLayout.ThisPartition),
			fmt.Sprintf("The byte offset %v, did not match the this partition value %v", l.TotalByteCount, partitionLayout.ThisPartition))
	})

	seg.Test("Checking the previous partition pointer is the correct byte position", func() bool {
		return tester.Expect(uint64(l.current)).To(Equal(partitionLayout.PreviousPartition),
			fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", l.current, partitionLayout.PreviousPartition))
	})

	//  implement a test case here
	//	fmt.Println(partitionLayout.ThisPartition, l.TotalByteCount)
	l.Rip = append(l.Rip, RIP{byteOffset: uint64(l.TotalByteCount), sid: partitionLayout.BodySID})
	//	fmt.Println(klvItem.TotalLength())
	l.current = l.TotalByteCount
	l.TotalByteCount += klvItem.TotalLength()

	// flush out the header metadata
	// as it is not used yet (apart from the primer)
	flushedMeta := 0
	for flushedMeta < int(partitionLayout.HeaderByteCount) {
		flush, open := <-metadata

		if !open {
			return fmt.Errorf("Error when using klv data klv stream interrupted")
		}
		flushedMeta += flush.TotalLength()

	}
	l.TotalByteCount += flushedMeta
	//hoover up the indextable and remove it to prevent it being mistaken as essence
	if partitionLayout.IndexTable {
		index, open := <-metadata
		if !open {
			return fmt.Errorf("Error when using klv data klv stream interrupted")
		}
		l.TotalByteCount += index.TotalLength()
	}
	// position += md.currentContainer.HeaderLength
	seg.result()
	return nil
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
		//header
		partPack.PartitionType = "header"
	case 03:
		//body
		partPack.PartitionType = "body"
	case 04:
		//footer
		partPack.PartitionType = "footer"
	default:
		//is nothing
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

		//develop and index table body to use
		partPack.IndexTable = true
	}

	partPack.TotalHeaderLength = totalLength

	// partion extract returns the type of partition and the length.
	//	fmt.Println(partPack, "pack here")
	//	fmt.Println(partPack, totalLength, "My partition oack")
	return partPack
}

type RIP struct {
	sid        uint32
	byteOffset uint64
}

func (l *layout) ripHandle(rip *klv.KLV) {

	// check the positions it gives with the logged positions
	length, _ := klv.BerDecode(rip.Length)

	ripLength := length - 4

	var gotRip []RIP

	for i := 0; i < ripLength; i += 12 {
		gotRip = append(gotRip, RIP{sid: order.Uint32(rip.Value[i : i+4]), byteOffset: order.Uint64(rip.Value[i+4 : i+12])})
	}

	//	testing.T
	// GinkgoWriter
	// var t *testing.T
	//	defer GinkgoRecover()
	//RegisterFailHandler(Fail)
	seg := newSegmentTest(l.testLog, "Random Index Pack Tests")
	res := NewGomegaWithT(seg)

	// res.Expect()

	seg.Test("Checking the partition positions in the file match those in the supplied random index pack", func() bool {
		return res.Expect(l.Rip).To(Equal(gotRip), "The generated index pack did not match")
	})

	//fmt.Println(res.Expect(l.Rip).To(Equal(gotRip)), "some ingo", "more desc", "lots of stuff", 342)
	//fmt.Println(res.Expect(4).To(Equal(6), "some ingo"))

	//	fmt.Println("MIDDLE")

	// fmt.Println(gotRip)
	//Expect(l.Rip).To(Equal(gotRip))
	seg.result()
}

/*

use go convey assertions with wrapping in the testing.T method?
*/
