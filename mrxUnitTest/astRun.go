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

	fmt.Println(node.FindSymbols(mxf2go.LabelsRegister[mxf2go.DescriptiveMetadataTrack].UL))

	tester.segment.Header("Checking the descriptive metadata is present in the file ", func() {

		descriptives := node.FindSymbol(mxf2go.LabelsRegister[mxf2go.DescriptiveMetadataTrack].UL)
		tester.segment.Test("Checking the descriptive metadata is present in the file ", func() bool {
			return tester.Expect(descriptives).ToNot(BeNil())
		})
		//resTrack := p.FindSymbol(nil, mxf2go.DescriptiveMetadataTrack) // look through the standards you out a test in
		// find syntax for starting at the route

		resFramework := descriptives.FindSymbol(mxf2go.LabelsRegister[mxf2go.MXFTextBasedFramework].UL)
		fmt.Println(resFramework)
		tester.segment.Test("Checking the descriptive next bit is present in the file ", func() bool {
			return tester.Expect(descriptives).ToNot(BeNil())
		})
		//	resIds := p.FindSymbols(resFramework, mrx2go.MetarexID, mrx2go.ExtraID)

		// check the shalls,
		// then check the behaviour
		//	tester.Expect(resTrack).ToNot(BeNil())
		//	tester.Expect(len(resIds)).ToNot(BeNil())
		//	tester.Expect(resFramework).

	})
}

/*

func mrxEmbeddedTimedDocuments(node *Node) {
	// find the st310 contexts
	ctx := node.FindSymbola(nil, st310 context)

	tester := newTester(os.Stdout, fmt.Sprintf("Embedded document tests", "delete later"))
	defer tester.Result()

	// check the 2057 document is there
	documentCount := ctx.FindSymbol(mxf2go.RP2057DocCount)

	// make a small loop to find the contexts ndocuments that I'm looking for out of this
	// 2057 partition. MRX path within the go framework.
	// Keep it metarex friendly

	tester.segment.Test("Checking the descriptive metadata is present in the file ", func() bool {
		resTrack := p.FindSymbol(nil, mxf2go.DescriptiveMetadataTrack) // look through the standards you out a test in
		// find syntax for starting at the route

		resFramework := p.FindSymbol(resTrack, mrx2go.MetarexDMFramework)

		resIds := p.FindSymbols(resFramework, mrx2go.MetarexID, mrx2go.ExtraID)


		// check the shalls,
		// then check the behaviour
		tester.Expect(resTrack).ToNot(BeNil())
		tester.Expect(len(resIds)).ToNot(BeNil())
		tester.Expect(resFramework).
		return tester.Expect(1).To(Equal(1),
		//ctx.URI
			fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", actualPrevPosition, mp.PreviousPartition))
	})


}*/

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
	recurseSearch(f, ast, PartitionWalk, &PartitionTest{})

	mrxDescriptiveMD(ast)
	return nil

}

/*
partition test
would be an object that logs the results from the previous
partition


Essence test would also be
keeping track of the essence keys
*/

type TestNode interface {
	TestNode(stream io.ReadSeeker, node *Node)

	// embed the functions
}

// add some search parameters in
// walk functions
// talk about how we organise things
func recurseSearch(stream io.ReadSeeker, node *Node, walker func(n *Node) bool, tn TestNode) {
	if node == nil {
		return
	}

	for _, n := range node.Children {
		if walker(n) {
			tn.TestNode(stream, n)
		}
		// search its children regardless
		recurseSearch(stream, n, walker, tn)
	}
}

func PartitionWalk(n *Node) bool {
	if n == nil {
		return false
	}
	if _, ok := n.Properties.(PartitionProperties); ok {
		// create an object that tests all the partitoins e.g. logs information for that run like
		// previous positiuons

		return ok
	}

	return false
}

type PartitionTest struct {
	Partitions []RIP
}

func (pt *PartitionTest) TestNode(stream io.ReadSeeker, node *Node) {

	partProps := node.Properties.(PartitionProperties)
	// this will only take partitions
	tester := newTester(os.Stdout, fmt.Sprintf("Partition %0d Tests", partProps.PartitionCount))
	defer tester.Result()
	// run all the parition tests
	// make a KLV extraction func
	k := nodeToKLV(stream, node)
	mp := partitionExtract(k)

	if partProps.PartitionType != "RIP" {

		actualPrevPosition := uint64(0)
		if len(pt.Partitions) > 0 {
			actualPrevPosition = uint64(pt.Partitions[len(pt.Partitions)-1].byteOffset)
		}
		fmt.Println(pt, node)
		tester.segment.Test("Checking the previous partition pointer is the correct byte position", func() bool {
			return tester.Expect(actualPrevPosition).To(Equal(mp.PreviousPartition),
				fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", actualPrevPosition, mp.PreviousPartition))
		})

		tester.segment.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
			return tester.Expect(uint64(node.Key.Start)).To(Equal(mp.ThisPartition),
				fmt.Sprintf("The byte offset %v, did not match the this partition value %v", node.Key.Start, mp.ThisPartition))
		})

		pt.Partitions = append(pt.Partitions, RIP{byteOffset: uint64(node.Key.Start), sid: mp.BodySID})
	} else {
		length, _ := klv.BerDecode(k.Length)

		ripLength := length - 4

		var gotRip []RIP

		for i := 0; i < ripLength; i += 12 {
			gotRip = append(gotRip, RIP{sid: order.Uint32(k.Value[i : i+4]), byteOffset: order.Uint64(k.Value[i+4 : i+12])})
		}

		tester.segment.Test("Checking the partition positions in the file match those in the supplied random index pack", func() bool {
			return tester.Expect(gotRip).To(Equal(pt.Partitions), "The generated index pack did not match the file index Pack")
		})
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
