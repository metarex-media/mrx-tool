package mrxUnitTest

import (
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/klv"
)

func MRXTest(doc io.ReadSeeker, w io.Writer, testspecs ...Specifications) error {

	klvChan := make(chan *klv.KLV, 1000)

	// get the specifications here
	testspecs = append(testspecs, NewISXD(), NewGeneric())

	// get an identical map of the base tests and
	// the skipped specifications.
	base, skips := generateSpecifications(testspecs...)

	// generate the AST, assigning the tests
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

			if len(part.HeaderMetadata) > 0 {
				// delete the map key for tests of this type
				delete(skips.Part, HeaderKey)

				tc.Header(fmt.Sprintf("testing header metadata of a %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {

					for _, child := range part.HeaderMetadata {
						testChildNodes(doc, child, part.Props.Primer, t, skips)
					}
				})

				tc.Header(fmt.Sprintf("testing header properties of a %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
					for _, child := range part.Tests.tests {

						childer := *child
						childer(doc, part)(t)
						if !t.testPass() {
							part.FlagFail()
						}
					}
				})
			}
		//	tc.headerTest(doc, part, specifications...)
		case BodyPartition, GenericStreamPartition:
			// delete the skipped partition to prove it has run
			if part.Props.PartitionType == BodyPartition {
				delete(skips.Part, EssenceKey)
			} else {
				delete(skips.Part, GenericKey)
			}

			tc.Header(fmt.Sprintf("testing essence properties at %s partition at offset %v", part.Props.PartitionType, part.Key.Start), func(t Test) {
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

	// check for any left over keys in
	if len(skips.Node) > 0 {
		for k := range skips.Node {
			tc.RegisterSkippedTest(k, "a skipped node test")
		}
	}

	if len(skips.Part) > 0 {
		for k := range skips.Part {
			tc.RegisterSkippedTest(k, "a skipped partition test")
		}

	}

	return nil
}

func generateSpecifications(testspecs ...Specifications) (base, skips Specifications) {
	base = Specifications{Node: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)),
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

	skips = Specifications{Node: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)),
		Part: make(map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode) func(t Test)),
	}

	for k, v := range base.Node {
		skips.Node[k] = v
	}

	for k, v := range base.Part {
		skips.Part[k] = v
	}

	return base, skips
}

// testChildNodes run any tests on the metadata and their children
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
