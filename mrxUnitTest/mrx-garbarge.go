package mrxUnitTest

/*
import (
	"fmt"
	"os"

	. "github.com/onsi/gomega"
) */

/*
func mrxDescriptiveMD() {

	tester := newTester(os.Stdout, fmt.Sprintf("Partition %0d Tests", "delete later"))
	defer tester.Result()

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
			fmt.Sprintf("The previous partition at %v, did not match the declared previous partition value %v", actualPrevPosition, mp.PreviousPartition))
	})

	/*
*/
/*
}

// Locate a text docuemnt in a SMPTE st310 partition (RP2057)
func mrxEmbeddedTimedDocuemntes() {
	ctx := p.FindContext(nil, st310 context)

	tester := newTester(os.Stdout, fmt.Sprintf("Partition %0d Tests", "delete later"))
	defer tester.Result()

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


}

*/
