package mrxUnitTest

import (
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/klv"
)

/*

test output looks like

how are tests built
they really focus on each segment - some things rely on the knowledge of the content

run order?

types of test:
 - partition
	- header meta
	- index
 - essence
 - structural - partitions in the right place

 some tests are not standalone e.g. does the headermetadata metch the partition count

API design

SpecInformation() Fullpath to the test for ech test

test("message", "spec details", {
	"Expect"
})


set up as the AST tree is found

Node and test carries everything

testing header
	testing against ISXD spec
		5.1 x is this
		pass/fail


*/

func genSpec(docName, sections, command string, commandPosition int) string {
	// is there a parent required
	return fmt.Sprintf("%s%s%s%v", docName, sections, command, commandPosition)
}

// Specification are the test functions for testing an MXF file to
// a specification
type Specification interface {
	// Test the header partition for metadata etc
	TestHeader(doc io.ReadSeeker, header *PartitionNode) func(t Test)
	// TestEssence for testing the essence within a partition
	TestEssence(doc io.ReadSeeker, header *PartitionNode) func(t Test)
	// test the overall structure of the mxf file
	TestStructure(doc io.ReadSeeker, mxf *MXFNode) func(t Test)
	// TestExtra is for any test cases not covered by, header, essence or structure tests
	TestExtra(doc io.ReadSeeker, mxf *MXFNode) func(t Test)
	// TestMarker() - something the test has to find in order to run
}

func MRXTest(doc io.ReadSeeker, w io.Writer, specifications ...Specification) error {

	klvChan := make(chan *klv.KLV, 1000)

	ast, genErr := MakeAST(doc, klvChan, 10)

	if genErr != nil {
		return genErr
	}
	// Test Structure

	/// go through each partition
	// switch test header/footer
	// test essence
	// test generic

	// testStructure
	tc := NewTestContext(w)
	defer tc.EndTest()

	tc.structureTest(doc, ast, specifications...)

	for _, part := range ast.Partitions {

		// check the essence in each partitoin?
		switch part.Props.PartitionType {
		case HeaderPartition, FooterPartition:
			tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
			tc.essTest(doc, part, specifications...)
		case RIPPartition:
			// not sure what happens here yet
		}
	}

	tc.extraTest(doc, ast, specifications...)

	return nil
}

func (tc *TestContext) structureTest(doc io.ReadSeeker, mxf *MXFNode, specifications ...Specification) {

	tc.Header("testing mxf file structure", func(t Test) {
		for _, spec := range specifications {
			spec.TestStructure(doc, mxf)(t)
		}
	})
}

func (tc *TestContext) extraTest(doc io.ReadSeeker, mxf *MXFNode, specifications ...Specification) {

	tc.Header("testing mxf file structure", func(t Test) {
		for _, spec := range specifications {
			spec.TestExtra(doc, mxf)(t)
		}
	})
}

func (tc *TestContext) headerTest(doc io.ReadSeeker, header *PartitionNode, specifications ...Specification) {

	tc.Header(fmt.Sprintf("testing header %s partition at offset %v", header.Props.PartitionType, header.Key.Start), func(t Test) {

		for _, spec := range specifications {

			spec.TestHeader(doc, header)(t)
		}

	})
}

func (tc *TestContext) essTest(doc io.ReadSeeker, header *PartitionNode, specifications ...Specification) {

	tc.Header(fmt.Sprintf("testing essence %s partition at offset %v", header.Props.PartitionType, header.Key.Start), func(t Test) {

		for _, spec := range specifications {
			spec.TestEssence(doc, header)(t)
		}

	})
}

// make the testContext a single body that holds the node?
// TestAnd Node
