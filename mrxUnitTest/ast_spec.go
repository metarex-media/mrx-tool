package mrxUnitTest

import (
	"fmt"
	"io"
	"os"

	"github.com/metarex-media/mrx-tool/klv"
	"gopkg.in/yaml.v3"
)

type demo struct {
	partitions     []demoTest
	essTests       []demoTest
	structureTests []demoTest
}

type demoTest struct {
	spec   SpecDetails
	Test   func()
	Marker string
}

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

	// get the specifications here
	testspecs := []SpecTests{NewISXD()}

	base := SpecTests{Node: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)),
		Part: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode, primer map[string]string) func(t Test)),
		MXF:  make([]*func(), 0)}

	for _, ts := range testspecs {
		for key, n := range ts.Node {
			out, ok := base.Node[key]
			if !ok {
				base.Node[key] = n
			} else {
				out = append(out, n...)
				base.Node[key] = out
			}

		}

		for key, n := range ts.Part {
			out, ok := base.Part[key]
			if !ok {
				base.Part[key] = n
			} else {
				out = append(out, n...)
				base.Part[key] = out
			}

		}
	}

	ast, genErr := MakeAST(doc, klvChan, 10, base)

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

	// load in default of 377 checker etc
	tc.Header("testing mxf file structure", func(t Test) {
		for _, structure := range ast.Tests.tests {
			str := *structure
			str(doc, ast, nil)(t)
		}
	})

	// 	tc.structureTest(doc, ast, specifications...)

	for _, part := range ast.Partitions {

		// check the essence in each partitoin?
		switch part.Props.PartitionType {
		case HeaderPartition, FooterPartition:
			tc.Header(fmt.Sprintf("testing header metadata at %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {

				for _, child := range part.HeaderMetadata {
					childTests(doc, child, part.Props.Primer, t)
				}
			})

			tc.Header(fmt.Sprintf("testing header stuff %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, child := range part.Tests.tests {

					childer := *child
					childer(doc, part, part.Props.Primer)(t)
					if !t.testPass() {
						part.callBack()
					}
				}
			})
		//	tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
		//	tc.essTest(doc, part, specifications...)
		case RIPPartition:
			// not sure what happens here yet
		}

	}
	fmt.Println(ast.Tests)
	//	tc.extraTest(doc, ast, specifications...)

	f, _ := os.Create("tester0.yaml")
	b, _ := yaml.Marshal(ast)
	f.Write(b)

	return nil
}

func childTests(doc io.ReadSeeker, node *Node, primer map[string]string, t Test) {

	if node == nil {
		return
	}

	for _, tester := range node.Tests.tests {
		test := *tester
		test(doc, node, primer)(t)
		if !t.testPass() {
			node.callBack()
		}
	}

	for _, child := range node.Children {
		childTests(doc, child, primer, t)
	}
}

func (tc *TestContext) structureTest(doc io.ReadSeeker, mxf *MXFNode, specifications ...Specification) {

	tc.Header("testing mxf file structure", func(t Test) {
		for _, spec := range specifications {
			spec.TestStructure(doc, mxf)(t)
		}
	})
}

func (tc *TestContext) extraTest(doc io.ReadSeeker, mxf *MXFNode, specifications ...Specification) {
	tc.Header("extra mxf tests", func(t Test) {
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
