package mrxUnitTest

import (
	"fmt"
	"io"
	"os"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
)

func mrxDescriptiveMD(node *Node) {

	tester := newTester(os.Stdout, fmt.Sprintf("Partition %s Tests", "delete later"))
	defer tester.Result()

	//	fmt.Println(node.FindSymbols(mxf2go.LabelsRegister[mxf2go.DescriptiveMetadataTrack].UL))

	tester.segment.Header("Checking the descriptive metadata is present in the file ", func() {

		descriptives := node.FindTypes(mxf2go.LabelsRegister[mxf2go.DescriptiveMetadataTrack].UL[13:])
		tester.segment.Test("Checking the descriptive metadata is present in the file ", func() bool {
			return tester.Expect(descriptives).ToNot(BeNil())
		})
		//resTrack := p.FindSymbol(nil, mxf2go.DescriptiveMetadataTrack) // look through the standards you out a test in
		// find syntax for starting at the route
		/*
			resFramework := descriptives[0].FindSymbol(mxf2go.LabelsRegister[mxf2go.MXFTextBasedFramework].UL)

			tester.segment.Test("Checking the descriptive next bit is present in the file ", func() bool {
				return tester.Expect(resFramework).ToNot(BeNil())
			})*/
		//	resIds := p.FindSymbols(resFramework, mrx2go.MetarexID, mrx2go.ExtraID)

		// check the shalls,
		// then check the behaviour
		//	tester.Expect(resTrack).ToNot(BeNil())
		//	tester.Expect(len(resIds)).ToNot(BeNil())
		//	tester.Expect(resFramework).

	})
}

func mrxEmbeddedTimedDocuments(doc io.ReadSeeker, node *Node) {
	// find the st310 contexts
	genericStreams := node.FindSymbols(GenericStreamPartition)

	// run tests on the length value
	// fmt.Println(genericStreams)

	tester := newTester(os.Stdout, "Embedded document tests")
	defer tester.Result()

	for _, gs := range genericStreams {

		// check the 2057 document is there
		//documentCount := ctx.FindSymbol(mxf2go.RP2057DocCount)

		// make a small loop to find the contexts ndocuments that I'm looking for out of this
		// 2057 partition. MRX path within the go framework.
		// Keep it metarex friendly
		tester.segment.Header(fmt.Sprintf("Checking the generic partition values at byte offset %v", gs.Key.Start), func() {

			partKLV := nodeToKLV(doc, gs)
			mxfPartition := partitionExtract(partKLV)

			tester.segment.Test("Checking the value of the HeaderByteCount is set to zero", func() bool {
				return tester.Expect(mxfPartition.HeaderByteCount).To(Equal(uint64(0)),
					fmt.Sprintf("The expected header count of 0, did not match the this partition value %v", mxfPartition.HeaderByteCount))
			})

			tester.segment.Test("Checking the value of the IndexByteCount is set to zero", func() bool {
				return tester.Expect(mxfPartition.IndexByteCount).To(Equal(uint64(0)),
					fmt.Sprintf("The expected Index Byte Count of 0, did not match the this partition value %v", mxfPartition.IndexByteCount))
			})

			tester.segment.Test("Checking the value of the IndexSID is set to zero", func() bool {
				return tester.Expect(mxfPartition.IndexSID).To(Equal(uint32(0)),
					fmt.Sprintf("The expected Index SID of 0, did not match the this partition value %v", mxfPartition.IndexByteCount))
			})

			// @TODO tests to add
			// well thats missing
			// 060e2b34.01010105.01020210.02020000
			/*
				6.2.3 - 410
					- body offset
					- body SID
				6.2.1 - 2057
					- element key bytes
				7.1
					- look for the descriptive metadata elements

			*/

			// desc seatch - get the footer - header if not found
			// get the preface - URN for the preface
			// search for the desriptive set - which is not currently included.

		})
	}

}

func ASTTest(f io.ReadSeeker, fout io.Writer) error {
	klvChan := make(chan *klv.KLV, 1000)
	ast, genErr := MakeAST(f, fout, klvChan, 10)

	if genErr != nil {
		return genErr
	}
	/*
		once we make the AST we now have to use it.


		Search via properties, implement a walker?

		thought experiment about the current ones

		Rip pack open the mxf partitoin for each one generate the information
		extract the rip last

		KEys ones, open each key and see if the order is preserved in a body


		The walkers almost loop throught the nodes and do the things they want to, like each test is a walker
		through the map.binary

		it just needs to know what its looking for.


		any node functions to include
	*/
	// run the partition tests

	// @TODO create a context for running tests

	mrxPartLayout(f, ast)
	mrxDescriptiveMD(ast)
	mrxEmbeddedTimedDocuments(f, ast)

	// run the tests clean up here

	return nil

}

func mrxPartLayout(stream io.ReadSeeker, node *Node) {

	parts := node.FindTypes(PartitionType)

	partitions := make([]RIP, 0)

	// this will only take partitions
	tester := newTester(os.Stdout, "Testing partition structure")
	defer tester.Result()

	for _, part := range parts {
		pklv := nodeToKLV(stream, part)
		if part.Properties.Symbol() != RIPPartition {

			actualPrevPosition := uint64(0)
			if len(partitions) > 0 {
				actualPrevPosition = uint64(partitions[len(partitions)-1].byteOffset)
			}

			mp := partitionExtract(pklv)
			// fmt.Println(pt, node)
			tester.segment.Test("Checking the previous partition pointer is the correct byte position", func() bool {
				return tester.Expect(actualPrevPosition).To(Equal(mp.PreviousPartition),
					fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", actualPrevPosition, mp.PreviousPartition))
			})

			tester.segment.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
				return tester.Expect(uint64(part.Key.Start)).To(Equal(mp.ThisPartition),
					fmt.Sprintf("The byte offset %v, did not match the this partition value %v", node.Key.Start, mp.ThisPartition))
			})

			partitions = append(partitions, RIP{byteOffset: uint64(part.Key.Start), sid: mp.BodySID})
		} else {
			length, _ := klv.BerDecode(pklv.Length)

			ripLength := length - 4

			var gotRip []RIP

			for i := 0; i < ripLength; i += 12 {
				gotRip = append(gotRip, RIP{sid: order.Uint32(pklv.Value[i : i+4]), byteOffset: order.Uint64(pklv.Value[i+4 : i+12])})
			}

			tester.segment.Test("Checking the partition positions in the file match those in the supplied random index pack", func() bool {
				return tester.Expect(gotRip).To(Equal(partitions), "The generated index pack did not match the file index Pack")
			})
		}
	}
}

func nodeToKLV(stream io.ReadSeeker, node *Node) *klv.KLV {
	stream.Seek(int64(node.Key.Start), 0)
	key := make([]byte, node.Key.End-node.Key.Start)
	leng := make([]byte, node.Length.End-node.Length.Start)
	val := make([]byte, node.Value.End-node.Value.Start)
	stream.Read(key)
	stream.Read(leng)
	stream.Read(val)

	return &klv.KLV{Key: key, Length: leng, Value: val}
}
