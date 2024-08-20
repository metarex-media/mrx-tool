package mrxUnitTest

import (
	"bytes"
	"context"
	"fmt"

	"github.com/metarex-media/mrx-tool/klv"
	"golang.org/x/sync/errgroup"
)

/*

type GomegaMatcher interface {
	Match(actual interface{}) (success bool, err error)
	FailureMessage(actual interface{}) (message string)
	NegatedFailureMessage(actual interface{}) (message string)
}
*/

type groupMatcher struct {
	groupID string
}

// GroupMatcher creates an gomega matching body
// for finding if the group is present within a partition.
// The key name is as found in the MSPTE register that means any 7f values are contained
// are used.
func HeaderContainsGroup(tester Test, UL string, partition []byte) {

	tester.Test("Checking if the partition contains the group "+UL, SpecDetails{},
		tester.Expect(partition).To(&groupMatcher{groupID: "060e2b34.027f0101.0d010201.01050100"},
			"the group was not found"),
	)
}

func (matcher *groupMatcher) Match(actual interface{}) (success bool, err error) {
	response, ok := actual.([]byte)

	if !ok {
		return false, fmt.Errorf("GroupMatcher matcher expects a byte array")
	}

	if len(response) == 0 {
		return false, nil
	}

	buffer := make(chan *klv.KLV, 10)
	cb := context.Background()
	errs, _ := errgroup.WithContext(cb)
	errs.Go(func() error {
		return klv.StartKLVStream(bytes.NewBuffer(response), buffer, 10)
	})

	// scan the input bytes and get the positioning
	var expeBytes [16]byte
	fmt.Sscanf(matcher.groupID, "%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		&expeBytes[0], &expeBytes[1], &expeBytes[2], &expeBytes[3], &expeBytes[4], &expeBytes[5], &expeBytes[6], &expeBytes[7],
		&expeBytes[8], &expeBytes[9], &expeBytes[10], &expeBytes[11], &expeBytes[12], &expeBytes[13], &expeBytes[14], &expeBytes[15])

	var swapPos []int

	for i, eb := range expeBytes {
		if eb == 0x7f {
			swapPos = append(swapPos, i)
		}
	}

	var match bool
	errs.Go(func() error {

		// @TODO: stop the klv channel blocking if this go function returns early before
		// reading everything.
		// currently i empty the channel at the end to run everything.
		defer func() {
			_, klvOpen := <-buffer
			for klvOpen {
				_, klvOpen = <-buffer
			}
		}()

		// get the first bit of stream
		klvItem, klvOpen := <-buffer

		for klvOpen {
			for _, sp := range swapPos {
				klvItem.Key[sp] = 0x7f
			}
			if fullName(klvItem.Key) == matcher.groupID {
				match = true
				return nil
			}
			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}
		return nil
	})

	err = errs.Wait()
	if err != nil {
		return false, fmt.Errorf("failed to decode byte stream: %s", err.Error())
	}

	return match, nil
}

func (matcher *groupMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain the group ID of\n\t%#v", actual, matcher.groupID)
}

func (matcher *groupMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to contain the group ID of\n\t%#v", actual, matcher.groupID)
}
