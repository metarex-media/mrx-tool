package mrxUnitTest

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// NewTester generates a new tester for a segment of tests
func newTester(dest io.Writer, segmentHeader string) *CompleteTest {

	var log bytes.Buffer
	seg := &segmentTest{header: segmentHeader, errChannel: make(chan string, 5), testBuffer: log, log: dest}

	ct := &CompleteTest{
		segment: seg,
	}
	// initialise the gomega tester object
	ct.tester = NewWithT(seg)
	return ct

}

func (c *CompleteTest) Test(message string, assert func() bool) {
	c.segment.Test(message, assert)
}

func (c *CompleteTest) Result() {
	c.segment.result()
}

type CompleteTest struct {
	segment *segmentTest
	tester
}

type Test interface {
	Test(message string, assert func() bool)
	Expect(actual interface{}, extra ...interface{}) types.Assertion
}

// tester is a workaround to wrap the gomega/internal object
type tester interface {
	Expect(actual interface{}, extra ...interface{}) types.Assertion
}

// wrap the results for later
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

// Test runs the
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
