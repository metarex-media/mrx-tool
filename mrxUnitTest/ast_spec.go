package mrxUnitTest

import (
	"fmt"
	"io"
	"os"

	"github.com/metarex-media/mrx-tool/klv"
	"gopkg.in/yaml.v3"
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

func MRXTest(doc io.ReadSeeker, w io.Writer) error {

	klvChan := make(chan *klv.KLV, 1000)

	// get the specifications here
	testspecs := []Specifications{NewISXD(), NewGeneric()}

	base := Specifications{Node: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)),
		Part: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode) func(t Test)),
		MXF:  make([]*func(doc io.ReadSeeker, isxdDesc *MXFNode) func(t Test), 0)}

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

		base.MXF = append(base.MXF, ts.MXF...)
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
			str(doc, ast)(t)
		}
	})

	// 	tc.structureTest(doc, ast, specifications...)

	for _, part := range ast.Partitions {

		// check the essence in each partitoin?
		switch part.Props.PartitionType {
		case HeaderPartition, FooterPartition:
			tc.Header(fmt.Sprintf("testing header metadata at %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {

				for _, child := range part.HeaderMetadata {
					testChildNodes(doc, child, part.Props.Primer, t)
				}
			})

			tc.Header(fmt.Sprintf("testing header stuff %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, child := range part.Tests.tests {

					childer := *child
					childer(doc, part)(t)
					if !t.testPass() {
						part.callBack()
					}
				}
			})
		//	tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
			tc.Header(fmt.Sprintf("testing essence stuff %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, tests := range part.Tests.tests {

					test := *tests
					test(doc, part)(t)
					if !t.testPass() {
						part.callBack()
					}
				}
			})
		//	tc.essTest(doc, part, specifications...)
		case RIPPartition:
			// not sure what happens here yet
		}

	}
	//	tc.extraTest(doc, ast, specifications...)

	f, _ := os.Create("tester0.yaml")
	b, _ := yaml.Marshal(ast)
	f.Write(b)

	return nil
}

func testChildNodes(doc io.ReadSeeker, node *Node, primer map[string]string, t Test) {

	if node == nil {
		return
	}

	for _, tester := range node.Tests.testsWithPrimer {
		test := *tester
		test(doc, node, primer)(t)
		if !t.testPass() {
			node.callBack()
		}
	}

	for _, child := range node.Children {
		testChildNodes(doc, child, primer, t)
	}
}

// make the testContext a single body that holds the node?
// TestAnd Node
