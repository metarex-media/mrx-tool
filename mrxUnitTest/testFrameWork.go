package mrxUnitTest

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func NewTestContext(dest io.Writer) *TestContext {
	return &TestContext{w: dest, globalPass: true}
}

type TestContext struct {
	w          io.Writer
	globalPass bool
}

func (tc *TestContext) EndTest() {
	if tc.globalPass {
		tc.w.Write([]byte("ALL Tests PASSED"))
		return
	}

	tc.w.Write([]byte("Test Failed"))
}

// Header is a wrapper for the tests,
// Adding more context to the results
func (s *TestContext) Header(message string, tests func(t Test)) {

	var log bytes.Buffer
	seg := &segmentTest{header: message, errChannel: make(chan string, 5), testBuffer: log, log: s.w}

	ct := &CompleteTest{
		segment: seg,
	}
	// initialise the gomega tester object
	mid := NewWithT(seg)
	out := assertionWrapper{out: mid}
	ct.gomegaExpect = out

	defer ct.Result()

	log.Write([]byte(fmt.Sprintf("	%v\n", message)))
	tests(ct)

	if seg.failCount != 0 {
		s.globalPass = false
	}
}

// NewTester generates a new tester for a segment of tests
func newTester(dest io.Writer, segmentHeader string) *CompleteTest {

	var log bytes.Buffer
	seg := &segmentTest{header: segmentHeader, errChannel: make(chan string, 5), testBuffer: log, log: dest}

	ct := &CompleteTest{
		segment: seg,
	}
	// initialise the gomega tester object
	mid := NewWithT(seg)
	out := assertionWrapper{out: mid}
	ct.gomegaExpect = out
	return ct

}

func (c *CompleteTest) Test(message string, assert func() bool) {
	c.segment.Test(message, assert)
}

func (c *CompleteTest) Result() {
	c.segment.result()
}

func (c *CompleteTest) Fail() {

}

type Assertions interface {
	Shall(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool
	types.Assertion
}

// ExportAssertions wraps the basic types.assertions with some
// extra names to allow th eMXf library to be followed better
type ExportAssertions struct {
	standard types.Assertion
}

func (e ExportAssertions) Shall(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e ExportAssertions) ShallNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.ToNot(matcher, optionalDescription...)
}

func (e ExportAssertions) NotTo(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.NotTo(matcher, optionalDescription...)
}
func (e ExportAssertions) To(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e ExportAssertions) ToNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.ToNot(matcher, optionalDescription...)
}
func (e ExportAssertions) Should(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e ExportAssertions) ShouldNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e ExportAssertions) WithOffset(offset int) types.Assertion {
	return e.standard.WithOffset(offset)
}

func (e ExportAssertions) Error() types.Assertion {
	return e.standard.Error()
}

type assertionWrapper struct {
	out gomegaExpect
}

func (aw assertionWrapper) Expect(actual interface{}, extra ...interface{}) Assertions {
	return ExportAssertions{aw.out.Expect(actual, extra...)}
}

type gomegaExpect interface {
	Expect(actual interface{}, extra ...interface{}) types.Assertion
}

type CompleteTest struct {
	segment      *segmentTest
	gomegaExpect Expecter
	// tester       Tester
}

func (ct CompleteTest) Expect(actual interface{}, extra ...interface{}) Assertions {
	return ct.gomegaExpect.Expect(actual, extra...)
}

type Test interface {
	Tester
	Expecter
}

// Expecter is a workaround to wrap the gomega/internal object
type Expecter interface {
	Expect(actual interface{}, extra ...interface{}) Assertions
}

type Tester interface {
	Test(message string, assert func() bool)
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
	gap := "    "
	s.testBuffer.Write([]byte(fmt.Sprintf("	%s%v\n", gap, message)))
	if assert() {
		s.testBuffer.Write([]byte(fmt.Sprintf("        %sPass\n", gap)))
	} else {
		s.failCount++
		s.testBuffer.Write([]byte(fmt.Sprintf("        %sFail!", gap)))
		s.testBuffer.Write([]byte(fmt.Sprintf("%v\n", strings.ReplaceAll(<-s.errChannel, "\n", "\n            "+gap))))
	}
}

// Header is a wrapper for the tests,
// Adding more context to the results
func (s *segmentTest) Header(message string, tests func()) {
	// s.testCount++
	// log headers
	s.testBuffer.Write([]byte(fmt.Sprintf("	%v\n", message)))
	tests()
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
