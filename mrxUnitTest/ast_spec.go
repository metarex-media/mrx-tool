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

	skips := Specifications{Node: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)),
		Part: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode) func(t Test)),
	}

	for k, v := range base.Node {
		skips.Node[k] = v
	}

	for k, v := range base.Part {
		skips.Part[k] = v
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
			// delete the map key for tests of this type
			delete(skips.Part, HeaderKey)

			tc.Header(fmt.Sprintf("testing header metadata at %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {

				for _, child := range part.HeaderMetadata {
					testChildNodes(doc, child, part.Props.Primer, t, skips)
				}
			})

			tc.Header(fmt.Sprintf("testing header stuff %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, child := range part.Tests.tests {

					childer := *child
					childer(doc, part)(t)
					if !t.testPass() {
						part.FlagFail()
					}
				}
			})
		//	tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
			if part.Props.PartitionType == BodyPartition {
				delete(skips.Part, EssenceKey)
			} else {
				delete(skips.Part, GenericKey)
			}

			tc.Header(fmt.Sprintf("testing essence stuff %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
				for _, tests := range part.Tests.tests {

					test := *tests
					test(doc, part)(t)
					if !t.testPass() {
						part.FlagFail()
					}
				}
			})
		//	tc.essTest(doc, part, specifications...)
		case RIPPartition:
			// not sure what happens here yet
		}

	}
	//	tc.extraTest(doc, ast, specifications...)

	if len(skips.Node) > 0 {
		skip := map[string][]string{"Skipped tests for the following  ULs": make([]string, len(skips.Node))}
		i := 0
		for k := range skips.Node {
			skip["Skipped tests for the following  ULs"][i] = k
			i++
		}

		skipBytes, _ := yaml.Marshal(skip)
		_, err := w.Write(skipBytes)
		if err != nil {
			return err
		}
	}

	if len(skips.Part) > 0 {
		skip := map[string][]string{"Skipped tests for the following partitions": make([]string, len(skips.Part))}
		i := 0
		for k := range skips.Part {
			skip["Skipped tests for the following partitions"][i] = k
			i++
		}
		skipBytes, _ := yaml.Marshal(skip)
		_, err := w.Write(skipBytes)
		if err != nil {
			return err
		}
	}

	f, _ := os.Create("tester0.yaml")
	b, _ := yaml.Marshal(ast)
	f.Write(b)

	return nil
}

type skipped struct {
	Field  string
	Missed []string
}

func testChildNodes(doc io.ReadSeeker, node *Node, primer map[string]string, t Test, skips Specifications) {

	if node == nil {
		return
	}

	for _, tester := range node.Tests.testsWithPrimer {
		delete(skips.Node, node.Properties.UL())
		test := *tester
		test(doc, node, primer)(t)
		if !t.testPass() {
			node.FlagFail()
		}
	}

	for _, child := range node.Children {
		testChildNodes(doc, child, primer, t, skips)
	}
}

// make the testContext a single body that holds the node?
// TestAnd Node
