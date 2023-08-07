package mrxUnitTest

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/metarex-media/mrx-tool/klv"
	. "github.com/onsi/gomega"
)

var embeddedTextKey = [16]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0000, 0, 0, 0}
var embeddedBinaryKey = [16]byte{06, 0x0e, 0x2b, 0x34, 01, 01, 01, 0x0c, 0x0d, 01, 05, 0b1101, 0b0001, 0, 0, 0}
var binaryClockedKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x01, 0x0f, 0x02, 0x01, 0x01, 0x01, 0x7f, 0x00, 0x7f}
var textClockedKey = [16]byte{0x06, 0x0E, 0x2B, 0x34, 0x01, 0x02, 0x01, 0x05, 0x0e, 0x09, 0x05, 0x02, 0x01, 0x7f, 0x01, 0x7f}

type essenceCache struct {
	keys [][]byte
}

func (l *layout) essenceCheck(ess *klv.KLV) {
	// check the key
	// check the key matches the partition type - so clip wrapped should be in a stream partition

	// do the stashing then run all the checks afterwards
	l.cache.keys = append(l.cache.keys, ess.Key)

	// process the layout dynamically to reduce the need for post processing in the test

}

type pattern struct {
	pattern [][]byte
	length  int
}

type presentKey struct {
	clockBinary, clockFrame []int
}

// implement keys here
// remove the need for an essence tag
func (l *layout) essenceTests() {

	if len(l.cache.keys) == 0 {
		return // there's no essence to test!
	}

	tester := newTester(l.testLog, fmt.Sprintf("Partition %0d Essence Tests", len(l.Rip)-1))
	defer tester.Result()

	// run generic tests first:
	/*

		keys repeat in the smae way and the numbers match what they are supposed to.
		test the first frame then

	*/
	pattern := getPattern(l.cache.keys)
	tester.TestEssenceKeyFramePattern(pattern, l.cache.keys)
	// check the keys contain the correct element counts etc
	// run an individual key checker on the pattern
	/*
		for essKey := range pattern {
			if key matches a metarex key then check
		}
	*/

	// the case statment to the specific type of partiton

	// if the current partition is a stream check the key length should be 1
	// and that the key is of a correct type

	//test.Test("Checking the this partition pointer matches the actual byte offset of the file", func() bool {
	//		return tester.Expect(uint64(l.TotalByteCount)).To(Equal(partitionLayout.ThisPartition),
	//			fmt.Sprintf("The byte offset %v, did not match the this partition value %v", l.TotalByteCount, partitionLayout.ThisPartition))
	//	})

	// check the keys are assigned to the right partition
	// ensure there's no mix

	// check the pattern for the moment
	/*
		pattern check algorithim
		if key is a metarex ID - check that it is in the right header
		check the element ad count. Add messages to the fail bit
		fail message to be dynamically sonctructed

		have a struct that tracks this for a pattern

		update these bits
		error message := pattern = has element count(2) should be 3 has position 01 should be 1
	*/
	tester.TestEssenceKeyLayouts(pattern)

	// @TODO insert more elements tests
	// loop through the keys and ensure they match the partition type ignoring the unknown
	//figure out if we want to cove them

	// are there any exact tests in the
	switch l.currentPartition.PartitionType {
	case bodyPartition:
	case genericStreamPartition:
		/*
			check the length if more than one bit is found, not illegal
		*/
	default:
		// do nothing
	}
	// reset the cache at the end
	l.cache.keys = make([][]byte, 0)
}

// TestEssenceKeyFramePattern checks the key order in the initial frame are repeated throughout the
// partition
func (c *CompleteTest) TestEssenceKeyFramePattern(pattern pattern, keys [][]byte) {
	allMatch := true
	for keyPos, key := range keys {
		if !reflect.DeepEqual(pattern.pattern[keyPos%pattern.length], key) {
			allMatch = false
		}
	}

	c.segment.Test("Checking the essence keys do not change order", func() bool {
		return c.t.Expect(allMatch).To(BeTrue(),
			fmt.Sprintf("The essence keys deviate from their original pattern of %s", "xyz"))
	})
}

// TestEssenceKeyLayouts checks the structure of the metarex essence keys.
// ensuring that the element count etc is preserved.
func (c *CompleteTest) TestEssenceKeyLayouts(pattern pattern) {
	errMessage := ""
	fail := false
	// embedded and clocked data
	Pos := 0
	// process chunks of elements at the time
	for Pos < len(pattern.pattern) {

		key := pattern.pattern[Pos]

		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		keyCopy[13], keyCopy[15] = 0x7f, 0x7f
		// TODO split
		if bytes.Equal(keyCopy, binaryClockedKey[:]) || bytes.Equal(keyCopy, textClockedKey[:]) {
			if Pos == 0 && key[13] != 1 { //first element must have a count of 1
				fail = true
				errMessage += fmt.Sprintf("The first clocked element must have an element count of 1, received a value of %v for %s\n", key[13], fullName(key))
			}

			// @TODO inlcude a 0 bit as then the count is wrong
			//
			count := int(key[13])
			checkPos := 1
			for checkPos < count {
				var nextKey []byte
				// fence the array lengths
				if len(pattern.pattern) < Pos+checkPos-1 {
					nextKey = pattern.pattern[Pos+checkPos]
				} else {
					nextKey = []byte("a string to made to fail")
				}

				if !bytes.Equal(key, nextKey) {
					errMessage += fmt.Sprintf("Expected an element count of %v only got %v for %s\n", key[13], checkPos, fullName(key))

					break
				}
				checkPos++

			}
			Pos += checkPos
		} else {
			Pos++
		} /*
			else if bytes.Equal(key, embeddedTextKey[:]) || bytes.Equal(key, embeddedBinaryKey[:]) {

			}*/
	}

	c.segment.Test("Checking the metarex essence keys have the correct element number and count", func() bool {
		return c.t.Expect(fail).To(BeFalse(),
			errMessage)
	})
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
