package mrxUnitTest

import (
	"fmt"
	"io"
	"os"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
)

func validISXD(doc io.ReadSeeker, node *MXFNode, tc *TestContext) {
	// set up comments for each test and check how it goes

	// XML parser name space etc - skip those

	// check the static track has points to every xml file

	// generic partitions should be ordered after the rest of the essence

	// ISXD seqeunce elements - read 2067 to find out what these are

	// check for frame wrapping - reread 379
}

func mrxDescriptiveMD(node *MXFNode, tc *TestContext) {

	//	fmt.Println(node.FindSymbols(mxf2go.LabelsRegister[mxf2go.DescriptiveMetadataTrack].UL))
	for _, mdnode := range node.Partitions {

		if mdnode.Props.PartitionType == HeaderPartition || mdnode.Props.PartitionType == FooterPartition {
			tc.Header(fmt.Sprintf("Checking the descriptive metadata is present in the file in the %s", mdnode.Props.PartitionType), func(t Test) {
				descriptives := make([]*Node, 0)
				for _, md := range mdnode.HeaderMetadata {
					descriptives = append(descriptives, md.FindTypes(mxf2go.LabelsRegister[mxf2go.DescriptiveMetadataTrack].UL[13:])...)
				}
				t.Test("Checking the descriptive metadata is present in the file ", func() bool {
					return t.Expect(descriptives).ToNot(BeNil())
				})

				for _, d := range descriptives {
					textFramework := d.FindSymbol("060e2b34.027f0101.0d010401.04010100")

					t.Test("Checking the descriptive metadata points to a Text based framework", func() bool {
						return t.Expect(textFramework).ToNot(BeNil())
					})

					textObj := d.FindSymbol("060e2b34.027f0101.0d010401.04020100")
					t.Test("Checking the text based framework points to a text based object set", func() bool {
						return t.Expect(textObj).ToNot(BeNil())
					})
				}
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
	}
}

func mrxEmbeddedTimedDocuments(doc io.ReadSeeker, node *MXFNode, tc *TestContext) {
	// find the st310 contexts
	// genericStreams := node.FindSymbols(GenericStreamPartition)

	// run tests on the length value
	// fmt.Println(genericStreams)

	for _, gs := range node.Partitions {
		if gs.Props.PartitionType == GenericStreamPartition {
			// check the 2057 document is there
			//documentCount := ctx.FindSymbol(mxf2go.RP2057DocCount)

			// make a small loop to find the contexts ndocuments that I'm looking for out of this
			// 2057 partition. MRX path within the go framework.
			// Keep it metarex friendly
			tc.Header(fmt.Sprintf("Checking the generic partition values at byte offset %v", gs.Key.Start), func(t Test) {

				partKLV := nodeToKLV(doc, &Node{Key: gs.Key, Length: gs.Length, Value: gs.Value})
				mxfPartition := partitionExtract(partKLV)

				t.Test("Checking the value of the HeaderByteCount is set to zero", func() bool {
					return t.Expect(mxfPartition.HeaderByteCount).Shall(Equal(uint64(0)),
						fmt.Sprintf("The expected header count of 0, did not match the this partition value %v", mxfPartition.HeaderByteCount))
				})

				t.Test("Checking the value of the IndexByteCount is set to zero", func() bool {
					return t.Expect(mxfPartition.IndexByteCount).To(Equal(uint64(0)),
						fmt.Sprintf("The expected Index Byte Count of 0, did not match the this partition value %v", mxfPartition.IndexByteCount))
				})

				t.Test("Checking the value of the IndexSID is set to zero", func() bool {
					return t.Expect(mxfPartition.IndexSID).To(Equal(uint32(0)),
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
	fo, _ := os.Create("out.log")
	// @TODO create a context for running tests
	tc := NewTestContext(fo)
	defer tc.EndTest()

	mrxPartLayout(f, ast, tc)
	mrxDescriptiveMD(ast, tc)
	mrxEmbeddedTimedDocuments(f, ast, tc)

	// run the tests clean up here

	return nil

}

func mrxPartLayout(stream io.ReadSeeker, node *MXFNode, tc *TestContext) {

	parts := node.Partitions

	partitions := make([]RIP, 0)

	for i, part := range parts {

		pklv := nodeToKLV(stream, &Node{Key: part.Key, Length: part.Length, Value: part.Value})
		if part.Props.Symbol() != RIPPartition {

			actualPrevPosition := uint64(0)
			if len(partitions) > 0 {
				actualPrevPosition = uint64(partitions[len(partitions)-1].byteOffset)
			}

			mp := partitionExtract(pklv)

			tc.Header(fmt.Sprintf("Testing partition %v", i), func(t Test) {

				// fmt.Println(pt, node)
				t.Test("Checking the previous partition pointer is the correct byte position", func() bool {
					return t.Expect(actualPrevPosition).To(Equal(mp.PreviousPartition),
						fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", actualPrevPosition, mp.PreviousPartition))
				})

				t.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
					return t.Expect(uint64(part.Key.Start)).To(Equal(mp.ThisPartition),
						fmt.Sprintf("The byte offset %v, did not match the this partition value %v", part.Key.Start, mp.ThisPartition))
				})
			})

			partitions = append(partitions, RIP{byteOffset: uint64(part.Key.Start), sid: mp.BodySID})
		} else {
			length, _ := klv.BerDecode(pklv.Length)

			ripLength := length - 4

			var gotRip []RIP

			for i := 0; i < ripLength; i += 12 {
				gotRip = append(gotRip, RIP{sid: order.Uint32(pklv.Value[i : i+4]), byteOffset: order.Uint64(pklv.Value[i+4 : i+12])})
			}
			tc.Header("Testing random index partition", func(t Test) {
				t.Test("Checking the partition positions in the file match those in the supplied random index pack", func() bool {
					return t.Expect(gotRip).To(Equal(partitions), "The generated index pack did not match the file index Pack")
				})
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
