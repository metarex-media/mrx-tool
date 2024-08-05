package mrxUnitTest

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func newTester(dest io.Writer, segmentHeader string) *CompleteTest {

	var log bytes.Buffer
	seg := &segmentTest{header: segmentHeader, errChannel: make(chan string, 5), testBuffer: log, log: dest}

	return &CompleteTest{
		segment: seg,
		t:       NewWithT(seg),
	}

}

func (c *CompleteTest) Result() {
	c.segment.result()
}

type CompleteTest struct {
	segment *segmentTest
	t       tester
}

// tester is a workaround to wrap the gomega/internal object
type tester interface {
	Expect(actual interface{}, extra ...interface{}) types.Assertion
}

/*
type mrxTest struct {
	// this will be a parent struct that handles all the different segments
	// or at least supplies the writer
}*/
/*
func newSegmentTest(dest io.Writer, segmentHeader string) *segmentTest {
	var log bytes.Buffer

	return &segmentTest{header: segmentHeader, errChannel: make(chan string, 5), testBuffer: log, log: dest}

}
*/
func (s *segmentTest) result() {

	s.log.Write([]byte(fmt.Sprintf("Running %s tests:\n", s.header)))
	s.log.Write(s.testBuffer.Bytes())
	s.log.Write([]byte(fmt.Sprintf("Ran %v tests: Passed:%v , Failed: %v\n", s.testCount, s.testCount-s.failCount, s.failCount)))

}

type segmentTest struct {
	header string
	// use the assertions to compare the error
	// generate a header
	// have an incremental counter of tests
	testCount int
	failCount int

	// segment pass

	// total test count
	errChannel chan string
	// testPass bool
	testBuffer bytes.Buffer
	log        io.Writer
}

// have a function to defer that does all the cleaning up when the tests

func (s *segmentTest) Test(message string, assert func() bool) {
	s.testCount++

	s.testBuffer.Write([]byte(fmt.Sprintf("	%v\n", message)))
	if assert() {
		s.testBuffer.Write([]byte("        Pass\n"))
	} else {
		s.failCount++
		s.testBuffer.Write([]byte("        Fail!"))
		s.testBuffer.Write([]byte(fmt.Sprintf("%v\n", strings.ReplaceAll(<-s.errChannel, "\n", "\n            "))))
	}
}

func (s *segmentTest) Fail() {
	// just don't do anything as each test is allowed to fail
}

func (s *segmentTest) Helper() {
	// leave as an empty call for the moment
}
func (s *segmentTest) Fatalf(format string, args ...interface{}) {

	// pipe the gomega err to be handled by the test wrapper
	s.errChannel <- fmt.Sprintf(format, args...)

}
