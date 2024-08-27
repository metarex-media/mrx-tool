package mrxUnitTest

import (
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/klv"
	. "github.com/onsi/gomega"
)

func NewGeneric() Specifications {

	ts := mrxPartLayout
	return Specifications{
		MXF: []*func(doc io.ReadSeeker, isxdDesc *MXFNode) func(t Test){&ts},
	}

}

const ST377Doc = "ST277-1:2019"

func mrxPartLayout(stream io.ReadSeeker, node *MXFNode) func(t Test) {

	parts := node.Partitions

	partitions := make([]RIP, 0)

	return func(t Test) {
		for _, part := range parts {

			pklv := nodeToKLV(stream, &Node{Key: part.Key, Length: part.Length, Value: part.Value})
			if part.Props.Symbol() != RIPPartition {

				actualPrevPosition := uint64(0)
				if len(partitions) > 0 {
					actualPrevPosition = uint64(partitions[len(partitions)-1].byteOffset)
				}

				mp := partitionExtract(pklv)

				// fmt.Println(pt, node)
				t.Test(fmt.Sprintf("Checking the previous partition pointer is the correct byte position for the %v partion at byte offset %v", part.Props.PartitionType, part.Key.Start),
					NewSpec(ST377Doc, "7.1", "Table5", 7),
					t.Expect(actualPrevPosition).To(Equal(mp.PreviousPartition),
						fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", actualPrevPosition, mp.PreviousPartition)),
				)

				t.Test(fmt.Sprintf("Checking the this partition pointer matches the actual byte offset of the file for the %v partion at byte offset %v", part.Props.PartitionType, part.Key.Start),
					NewSpec(ST377Doc, "7.1", "Table5", 8),
					t.Expect(uint64(part.Key.Start)).To(Equal(mp.ThisPartition),
						fmt.Sprintf("The byte offset %v, did not match the this partition value %v", part.Key.Start, mp.ThisPartition)))

				partitions = append(partitions, RIP{byteOffset: uint64(part.Key.Start), sid: mp.BodySID})
			} else {

				length, _ := klv.BerDecode(pklv.Length)
				ripLength := length - 4
				var gotRip []RIP
				for i := 0; i < ripLength; i += 12 {
					gotRip = append(gotRip, RIP{sid: order.Uint32(pklv.Value[i : i+4]), byteOffset: order.Uint64(pklv.Value[i+4 : i+12])})
				}

				t.Test("Checking the partition positions in the file match those in the supplied random index pack", NewSpec(ST377Doc, "12.2", "shall", 1),
					t.Expect(gotRip).To(Equal(partitions), "The generated index pack did not match the file index Pack"))

			}
		}
	}
}
