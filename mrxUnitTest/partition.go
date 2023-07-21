package mrxUnitTest

import (
	"encoding/binary"
	"fmt"

	"github.com/metarex-media/mrx-tool/klv"
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
	fmt.Println(partitionLayout.ThisPartition, l.TotalByteCount)
	fmt.Println(klvItem.TotalLength())
	l.TotalByteCount += klvItem.TotalLength()
	l.current = &partitionLayout

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
