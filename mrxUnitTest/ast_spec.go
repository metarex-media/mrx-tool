package mrxUnitTest

import (
	"fmt"
	"io"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
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

type tester struct {
	header []func(doc io.ReadSeeker, node *MXFNode, tc *TestContext)
}

type ISXD struct {
}

func (i ISXD) DocName() string {
	return "RDD-47:2018"
}

func genSpec(docName, sections, command string, commandPosition int) string {
	// is there a parent required
	return fmt.Sprintf("%s%s%s%v", docName, sections, command, commandPosition)
}
func (i ISXD) TestHeader(doc io.ReadSeeker, header *PartitionNode) func(t Test) {

	return func(t Test) {
		var isxdDesc *Node
		for _, child := range header.HeaderMetadata {
			isxdDesc = child.FindSymbol("060e2b34.02530105.0e090502.00000000")
			if isxdDesc != nil {
				break
			}
		}
		// rdd-47:2009/11.5.3/shall/4
		t.Test("Checking that the isxd descriptor is present in the header metadata"+genSpec(i.DocName(), "9.2", "shall", 1), func() bool {
			return t.Expect(isxdDesc).ShallNot(BeNil())
		})
		if isxdDesc != nil {
			// decode the group
			isxdDecode, err := DecodeGroupNode(doc, isxdDesc, header.Props.Primer)
			fmt.Println(isxdDecode, err)
			t.Test("Checking that the data essence coding filed is present in the isxd descriptor"+genSpec(i.DocName(), "9.3", "shall", 1), func() bool {
				return t.Expect(isxdDecode["DataEssenceCoding"]).Shall(Equal(mxf2go.TAUID{
					Data1: 101591860,
					Data2: 1025,
					Data3: 261,
					Data4: mxf2go.TUInt8Array8{14, 9, 6, 6, 0, 0, 0, 0},
				}))
			})
		}

		// handle the static track sections of the path
		GenericCountPositions := make([]int, 0)
		for i, part := range header.Parent.Partitions {
			// check the essence in each partitoin?
			if part.Props.PartitionType == GenericStreamPartition {
				// extra check is counting the steamIDs
				GenericCountPositions = append(GenericCountPositions, i)

			}
		}

		if len(GenericCountPositions) > 0 {
			// ibly run if there's any generic essence
			var staticTrack *Node

			for _, child := range header.HeaderMetadata {
				staticTrack = child.FindSymbol("060e2b34.027f0101.0d010101.01013a00")
				if staticTrack != nil {
					break
				}
			}

			t.Test("Checking that a static track is present in the header metadata "+genSpec(i.DocName(), "5.4", "shall", 1), func() bool {
				return t.Expect(staticTrack).ToNot(BeNil())
			})

			if staticTrack != nil {

				sequence := staticTrack.FindSymbol("060e2b34.027f0101.0d010101.01010f00")
				t.Test("Checking that the static track points to a sequence"+genSpec(i.DocName(), "5.4", "shall", 2), func() bool {
					return t.Expect(sequence).ToNot(BeNil())
				})

				t.Test("Checking that the static track sequence has as many sequence children as partitions"+genSpec(i.DocName(), "5.4", "shall", 2), func() bool {
					return t.Expect(len(sequence.Children)).Shall(Equal(len(GenericCountPositions)))
				})
			}
		}

	}
	// test ISXD descriptor

}

func (i ISXD) TestEssence(doc io.ReadSeeker, header *PartitionNode) func(t Test) {

	return func(t Test) {
		if header.Props.PartitionType == BodyPartition && len(header.Essence) > 0 {
			allISXD := true

			pattern := []string{}
			patternTally := true
			for _, e := range header.Essence {
				ess := nodeToKLV(doc, e)

				if fullNameMask(ess.Key, 13, 15) != "060e2b34.01020105.0e090502.017f017f" {
					allISXD = false
					break
				}
				fullKey := fullNameMask(ess.Key)
				if len(pattern) != 0 {
					if pattern[0] == fullKey {
						patternTally = false
					} else if patternTally {
						pattern = append(pattern, fullKey)
					}
				} else {
					pattern = append(pattern, fullKey)
				}

			}

			t.Test("Checking that the only ISXD essence keys are found in body partitions"+genSpec(i.DocName(), "7.5", "shall", 1), func() bool {
				return t.Expect(allISXD).Shall(BeTrue(), "Other essence keys found")
			})

			if allISXD {

				breakPoint := 0
				for i, e := range header.Essence {
					ess := nodeToKLV(doc, e)

					if fullNameMask(ess.Key) != pattern[i%len(pattern)] {
						breakPoint = e.Key.Start
						break
					}

				}

				t.Test("Checking that the content package order are regular throughout the essence stream"+genSpec(i.DocName(), "7.5", "shall", 1), func() bool {
					return t.Expect(breakPoint).Shall(Equal(0), fmt.Sprintf("irregular key found at byte offset %v", breakPoint))
				})
			}
		} else if header.Props.PartitionType == GenericStreamPartition {
			// check it passes 2057 rules
		}
	}
}

// specificions

type Specification interface {
	TestHeader(doc io.ReadSeeker, header *PartitionNode) func(t Test)
	TestEssence()
	TestStructure()
	TestExtra()
	// TestMarker() - something the test has to find in order to run
}

func MRXTest(doc io.ReadSeeker, w io.Writer, specifications ...Specification) error {

	klvChan := make(chan *klv.KLV, 1000)

	ast, genErr := MakeAST(doc, w, klvChan, 10)

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

	for _, part := range ast.Partitions {

		// check the essence in each partitoin?
		switch part.Props.PartitionType {
		case HeaderPartition, FooterPartition:
			tc.headerTest(doc, part)
		case BodyPartition, GenericStreamPartition:
			tc.essTest(doc, part)
		case RIPPartition:

		}
	}

	return nil
}

func (tc *TestContext) headerTest(doc io.ReadSeeker, header *PartitionNode, specifications ...Specification) {

	tc.Header(fmt.Sprintf("testing %s partition at offset %v", header.Props.PartitionType, header.Key.Start), func(t Test) {

		for _, spec := range specifications {
			fmt.Println(t, "T HERE")
			spec.TestHeader(doc, header)(t)
		}

	})
}

func (tc *TestContext) essTest(doc io.ReadSeeker, header *PartitionNode) {

	specifications := []func(doc io.ReadSeeker, header *PartitionNode) func(t Test){ISXD{}.TestEssence}

	tc.Header(fmt.Sprintf("testing %s partition at offset %v", header.Props.PartitionType, header.Key.Start), func(t Test) {

		for _, spec := range specifications {
			fmt.Println(t, "T HERE")
			spec(doc, header)(t)
		}

	})
}

// make the testContext a single body that holds the node?
// TestAnd Node
