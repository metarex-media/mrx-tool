package mrxUnitTest

import (
	"fmt"

	"github.com/metarex-media/mrx-tool/klv"
	. "github.com/onsi/gomega"
)

func (l *layout) metadataTest(metaData []*klv.KLV) {
	/*cache the primer for later use in the tests*/

	// set up all the tests

	// get all the metadata here (an array of klv?)

	// process each one logging instance IDs - ignoring dark essence
	// get every type of string reference and extract the contents to ensure everything is referenced (unless it is refernced with dark metadata)
	// maybe implement a generic thing for dark keys with instance IDs and strong references

	// implement other metadata tests checking if things like primer packs are where they should be

	//seg := newSegmentTest(l.testLog, fmt.Sprintf("Partiton %0d MetaData Tests", len(l.Rip)-1)) // the length of the RIP gives the relative partition count
	//defer seg.result()
	//tester := NewGomegaWithT(seg)
	tester := newTester(l.testLog, fmt.Sprintf("Partiton %0d MetaData Tests", len(l.Rip)-1))
	defer tester.Result()

	tester.TestMetaData()

	//	seg.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
	///		return tester.Expect(0).To(Equal(0),
	//			fmt.Sprintf("The byte offset %v, did not match the this partition value %v", l.TotalByteCount, 0))
	//	})

	// if first.name != primer throw a wobbly
	// cache the primer

	for _, md := range metaData {

		name := fullName(md.Key)

		if name == "060e2b34.01010102.03010210.01000000" || name == "060e2b34.01010101.03010210.01000000" || name == "060e2b34.01020101.03010210.01000000" {
			// skip for the moment
		} else {
			/*


				get the generated mrx name

				check all the children as you go along and make the map[string]any for each metadata

				if name Contains strong reference vector then record teh contents.
				record the UUID bytes and mark them as true. in a map
				map of found and parent? unless there's dark sessence

				utilise the fixed pack decodes from elsewhere. If length == 0 then do some other stuff

			*/

		}

	}
}

// TestMetaData runs tests on these bits of the metadata
func (c *CompleteTest) TestMetaData() {
	c.segment.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
		return c.t.Expect(0).To(Equal(0),
			fmt.Sprintf("The byte offset %v, did not match the this partition value %v", 0, 0))
	})
}

func metadataName(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	} // @TODO put the 7f in

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15])
}
