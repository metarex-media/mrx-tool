package mrxUnitTest

import (
	"fmt"
	"io"

	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
)

/*
keys = header, essence, UL, MXF
*/

const (
	HeaderKey = "header"
)

func NewISXD() SpecTests {
	nt := testISXDDescriptor
	f := fail

	gc := GenericCount
	return SpecTests{
		Node: map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test){
			"060e2b34.02530105.0e090502.00000000": {&nt, &f},
			//060e2b34.02530101.0d010101.01013b00
		},
		Part: map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode, primer map[string]string) func(t Test){
			HeaderKey: {&gc},
		},
	}

}
func fail(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test) {

	return func(t Test) {
		t.Test("Checking that the isxd descriptor is present in the header metadata", NewSpec("i.DocName()", "9.2", "shall", 1),
			t.Expect(5).Shall(BeNil()),
		)
	}
}
func testISXDDescriptor(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test) {

	return func(t Test) {

		// rdd-47:2009/11.5.3/shall/4
		t.Test("Checking that the isxd descriptor is present in the header metadata", NewSpec("i.DocName()", "9.2", "shall", 1),
			t.Expect(isxdDesc).ShallNot(BeNil()),
		)

		if isxdDesc != nil {
			// decode the group
			isxdDecode, err := DecodeGroupNode(doc, isxdDesc, primer)

			t.Test("Checking that the data essence coding filed is present in the isxd descriptor", NewSpec("i.DocName()", "9.3", "shall", 1),
				t.Expect(err).Shall(BeNil()),
				t.Expect(isxdDecode["DataEssenceCoding"]).Shall(Equal(mxf2go.TAUID{
					Data1: 101591860,
					Data2: 1025,
					Data3: 261,
					Data4: mxf2go.TUInt8Array8{14, 9, 6, 6, 0, 0, 0, 0},
				})))
		}
	}
}

func GenericCount(doc io.ReadSeeker, header *PartitionNode, _ map[string]string) func(t Test) {

	return func(t Test) {

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
				staticTrack = child.FindUL("060e2b34.027f0101.0d010101.01013a00")
				if staticTrack != nil {
					break
				}
			}

			t.Test("Checking that a static track is present in the header metadata ", NewSpec("i.DocName()", "5.4", "shall", 1),
				t.Expect(staticTrack).ToNot(BeNil()),
			)

			if staticTrack != nil {

				sequence := staticTrack.FindUL("060e2b34.027f0101.0d010101.01010f00")
				t.Test("Checking that the static track points to a sequence", NewSpec("i.DocName()", "5.4", "shall", 2),
					t.Expect(sequence).ToNot(BeNil()),
				)

				t.Test("Checking that the static track sequence has as many sequence children as partitions", NewSpec("i.DocName()", "5.4", "shall", 2),
					t.Expect(len(sequence.Children)).Shall(Equal(len(GenericCountPositions))),
				)
			}
		}

	}
	// test ISXD descriptor

}

type ISXD struct {
}

func (i ISXD) DocName() string {
	return "RDD-47:2018"
}

func (i ISXD) TestHeader(doc io.ReadSeeker, header *PartitionNode) func(t Test) {

	return func(t Test) {
		var isxdDesc *Node
		for _, child := range header.HeaderMetadata {
			isxdDesc = child.FindUL("060e2b34.02530105.0e090502.00000000")
			if isxdDesc != nil {
				break
			}
		}
		// rdd-47:2009/11.5.3/shall/4
		t.Test("Checking that the isxd descriptor is present in the header metadata", NewSpec(i.DocName(), "9.2", "shall", 1),
			t.Expect(isxdDesc).ShallNot(BeNil()),
		)

		if isxdDesc != nil {
			// decode the group
			isxdDecode, err := DecodeGroupNode(doc, isxdDesc, header.Props.Primer)

			t.Test("Checking that the data essence coding filed is present in the isxd descriptor", NewSpec(i.DocName(), "9.3", "shall", 1),
				t.Expect(err).Shall(BeNil()),
				t.Expect(isxdDecode["DataEssenceCoding"]).Shall(Equal(mxf2go.TAUID{
					Data1: 101591860,
					Data2: 1025,
					Data3: 261,
					Data4: mxf2go.TUInt8Array8{14, 9, 6, 6, 0, 0, 0, 0},
				})))
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
				staticTrack = child.FindUL("060e2b34.027f0101.0d010101.01013a00")
				if staticTrack != nil {
					break
				}
			}

			t.Test("Checking that a static track is present in the header metadata ", NewSpec(i.DocName(), "5.4", "shall", 1),
				t.Expect(staticTrack).ToNot(BeNil()),
			)

			if staticTrack != nil {

				sequence := staticTrack.FindUL("060e2b34.027f0101.0d010101.01010f00")
				t.Test("Checking that the static track points to a sequence", NewSpec(i.DocName(), "5.4", "shall", 2),
					t.Expect(sequence).ToNot(BeNil()),
				)

				t.Test("Checking that the static track sequence has as many sequence children as partitions", NewSpec(i.DocName(), "5.4", "shall", 2),
					t.Expect(len(sequence.Children)).Shall(Equal(len(GenericCountPositions))),
				)
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

			t.Test("Checking that the only ISXD essence keys are found in body partitions", NewSpec(i.DocName(), "7.5", "shall", 1),
				t.Expect(allISXD).Shall(BeTrue(), "Other essence keys found"),
			)

			if allISXD {

				breakPoint := 0
				for i, e := range header.Essence {
					ess := nodeToKLV(doc, e)

					if fullNameMask(ess.Key) != pattern[i%len(pattern)] {
						breakPoint = e.Key.Start
						break
					}

				}

				t.Test("Checking that the content package order are regular throughout the essence stream", NewSpec(i.DocName(), "7.5", "shall", 1),
					t.Expect(breakPoint).Shall(Equal(0), fmt.Sprintf("irregular key found at byte offset %v", breakPoint)),
				)
			}
		} else if header.Props.PartitionType == GenericStreamPartition {
			// check it passes 2057 rules
			// call 2057 properties?
			// which is then a 410
			headerKLV := nodeToKLV(doc, &Node{Key: header.Key, Length: header.Length, Value: header.Value})
			mp := partitionExtract(headerKLV)

			t.Test("Checking that the index byte count for the generic header is 0", NewSpec(i.DocName(), "7.5", "shall", 1),
				t.Expect(mp.IndexByteCount).Shall(Equal(uint64(0)), "index byte count not 0"),
			)

			t.Test("Checking that the header metadata byte count for the generic header is 0", NewSpec(i.DocName(), "7.5", "shall", 1),
				t.Expect(mp.HeaderByteCount).Shall(Equal(uint64(0)), "header metadata byte count not 0"),
			)

			t.Test("Checking that the index SID for the generic header is 0", NewSpec(i.DocName(), "7.5", "shall", 1),
				t.Expect(mp.IndexSID).Shall(Equal(uint32(0)), "index SID not 0"),
			)

			t.Test("checking the partition key meets the expected value of 060e2b34.02050101.0d010201.01031100", NewSpec(i.DocName(), "7.5", "shall", 1),
				t.Expect(fullName(headerKLV.Key)).Shall(Equal("060e2b34.02050101.0d010201.01031100")),
			)

			// the key has got to be the 410 key

			// key matches value of
			// 11 / 12 can be masked
			//060e2b34.0101010c.0d010509.01000000
			//"060e2b34.0101010c.0d010509.01000000"
			badKey := ""

			for _, ess := range header.Essence {
				essKLV := nodeToKLV(doc, ess)
				if fullNameMask(essKLV.Key, 11, 12) != "060e2b34.0101010c.0d01057f.7f000000" {
					badKey = fullName(essKLV.Key)
					break
				}
			}

			t.Test("checking the essence keys all have the value of 060e2b34.0101010c.0d01057f.7f000000", NewSpec(i.DocName(), "7.5", "shall", 1),
				t.Expect(badKey).Shall(Equal(""), "invalid key of "+badKey+" found"),
			)
		}
	}
}

// TestExtra returns an empty test
func (i ISXD) TestExtra(doc io.ReadSeeker, mxf *MXFNode) func(t Test) {
	return func(t Test) {}
}

func (i ISXD) TestStructure(doc io.ReadSeeker, mxf *MXFNode) func(t Test) {
	return func(t Test) {

		// find the generic paritions
		GenericCountPositions := make([]int, 0)
		var footerPresent, RIPPresent bool
		for i, part := range mxf.Partitions {

			// check the essence in each partitoin?
			switch part.Props.PartitionType {
			case BodyPartition:
			case GenericStreamPartition:
				// extra check is counting the steamIDs
				GenericCountPositions = append(GenericCountPositions, i)
			case FooterPartition:
				footerPresent = true
			case RIPPartition:
				RIPPresent = true
			}
		}

		endPos := len(mxf.Partitions)
		if footerPresent {
			endPos--
		}
		if RIPPresent {
			endPos--
		}

		expectedParts := make([]int, len(GenericCountPositions))
		for j := range expectedParts {
			expectedParts[j] = endPos - len(expectedParts) + j
		}
		t.Test("Checking that the generic partition positions match the expected positions at the end of the file", NewSpec(i.DocName(), "5.4", "shall", 3),
			t.Expect(expectedParts).Shall(Equal(GenericCountPositions)),
		)
	}
}
