package mrxUnitTest

import (
	"fmt"
	"reflect"

	"github.com/metarex-media/mrx-tool/klv"
	. "github.com/onsi/gomega"
)

type essenceCache struct {
	keys [][]byte
}

func (l *layout) essenceCheck(ess *klv.KLV) {
	// check the key
	// check the key matches the partition type - so clip wrapped should be in a stream partition

	// do the stashing then run all the checks afterwards
	l.cache.keys = append(l.cache.keys, ess.Key)

}

type pattern struct {
	pattern [][]byte
	length  int
}

func (l *layout) essenceTests() {
	test := newSegmentTest(l.testLog, fmt.Sprintf("Partiton %0d Essence Tests", len(l.Rip)))
	defer test.result()
	tester := NewGomegaWithT(test)

	// run generic tests first:
	/*

		keys repeat in the smae way and the numbers match what they are supposed to.
		test the first frame then

	*/
	pattern := getPattern(l.cache.keys)
	allMatch := true
	for keyPos, key := range l.cache.keys {
		if !reflect.DeepEqual(pattern.pattern[keyPos%pattern.length], key) {
			allMatch = false
		}
	}

	test.Test("Checking the essence keys do not change order", func() bool {
		return tester.Expect(allMatch).To(BeTrue(),
			fmt.Sprintf("The essence keys deviate from their original pattern of %s", "xyz"))
	})

	// the case statment to the specific type of partiton

	// if the current partition is a stream check the key length should be 1
	// and that the key is of a correct type

	//test.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
	//		return tester.Expect(uint64(l.TotalByteCount)).To(Equal(partitionLayout.ThisPartition),
	//			fmt.Sprintf("The byte offset %v, did not match the this partition value %v", l.TotalByteCount, partitionLayout.ThisPartition))
	//	})

	l.cache.keys = make([][]byte, 0)
}

func getPattern(keys [][]byte) pattern {
	if len(keys) == 0 {
		return pattern{}
	} else if len(keys) == 1 {
		return pattern{pattern: keys, length: 1}
	}

	base := pattern{pattern: make([][]byte, 0)}
	marker := keys[0]
	base.pattern = append(base.pattern, marker)

	var match bool

	for i, key := range keys[1:] {
		if reflect.DeepEqual(key, marker) {
			base.length = i + 1
			match = true
			break
		}
		base.pattern = append(base.pattern, key)

	}

	if !match {
		base.length = len(base.pattern)
	}

	return base

}
