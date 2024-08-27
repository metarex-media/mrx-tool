package mrxUnitTest

import (
	"fmt"
	"io"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"gopkg.in/yaml.v3"
)

// NewTestContext generates a new testContext that writes to des.
//
// ensure EndTest() is called to flush the results to the writer.
func NewTestContext(dest io.Writer) *TestContext {

	return &TestContext{w: dest, globalPass: true}
}

// TestContext is the global context for all the
// tests.
type TestContext struct {
	w          io.Writer
	globalPass bool
	report     Report
}

func (tc *TestContext) RegisterSkippedTest(key, desc string) {
	tc.report.SkippedTests = append(tc.report.SkippedTests, skippedTest{TestKey: key, Desc: desc})
}

// Report is the report structure of the
// MXF test report
type Report struct {
	// Did the overall test pass
	TestPass bool
	// the tests and their results
	Tests        []TestSection
	SkippedTests []skippedTest `yaml:"skippedTests,omitempty"`
}

type skippedTest struct {
	TestKey string
	Desc    string
}

type TestSection struct {
	// the header message for that batch of tests
	Header string
	// the tests themselves
	Tests []TestResult
	// the results
	Pass                 bool
	PassCount, FailCount int
}

type TestResult struct {
	Message string
	Checks  []check
}

type check struct {
	Pass       bool
	ErrMessage string `yaml:"errorMessage,omitempty"`
}

// End Test must be called to write the results to the io.Writer
func (tc *TestContext) EndTest() {
	if tc.globalPass {
		tc.report.TestPass = true
	}

	y, _ := yaml.Marshal(tc.report)
	tc.w.Write(y)

	// write the yaml and hanlde the error
}

// Header is a wrapper for the tests,
// Adding more context to the results.
// Header must be called to run the tests
func (s *TestContext) Header(message string, tests func(t Test)) {

	seg := &segmentTest{errChannel: make(chan string, 5), testPass: true, testReport: TestSection{Header: message, Tests: make([]TestResult, 0), Pass: true}}
	ct := &CompleteTest{
		segment: seg,
	}
	// initialise the gomega tester object
	mid := NewWithT(seg)
	out := assertionWrapper{out: mid}
	ct.gomegaExpect = out

	// run the tests
	tests(ct)

	if seg.failCount != 0 {
		s.globalPass = false
	}

	s.report.Tests = append(s.report.Tests, seg.testReport)
}

// Test runs the assertions and logs the resukts in the report
func (c *CompleteTest) Test(message string, specDetail SpecDetails, asserts ...bool) {
	c.segment.test(message, specDetail, asserts...)
}

// Test runs the test
func (s *segmentTest) test(message string, specDetail SpecDetails, asserts ...bool) {
	// update to catch the test without trying the function approach.
	// want multiple bits each conuting as a test
	s.testCount++
	//gap := "    "
	s.testPass = true
	// s.testBuffer.Write([]byte(fmt.Sprintf("	%s%s: %v\n", gap, specDetail, message)))

	te := TestResult{Message: fmt.Sprintf("%s: %v\n", specDetail, message), Checks: make([]check, len(asserts))}

	for i, assert := range asserts {
		if assert {
			te.Checks[i] = check{Pass: true}
			//	s.testBuffer.Write([]byte(fmt.Sprintf("        %sCheck %v Pass\n", gap, i)))
			s.testReport.PassCount++
		} else {
			s.testReport.FailCount++
			s.testReport.Pass = false

			s.testPass = false
			s.failCount++
			//	s.testBuffer.Write([]byte(fmt.Sprintf("        %sCheck %vFail!", gap, i)))
			select {
			case err := <-s.errChannel:
				// go from the first byte to stop it breaking the yaml layout with a 4- key
				te.Checks[i] = check{ErrMessage: fmt.Sprintf("%v", err[1:])}
			//	s.testBuffer.Write([]byte(fmt.Sprintf("%v\n", strings.ReplaceAll(err, "\n", "\n            "+gap))))
			default:
				panic("Gomega assertion not used for finding errors, aborting program, Must use syntax of t.Expect(val).Shall(BeNil())")
			}

		}
	}

	s.testReport.Tests = append(s.testReport.Tests, te)
}

type assertionWrapper struct {
	out gomegaExpect
}

func (aw assertionWrapper) Expect(actual interface{}, extra ...interface{}) Assertions {
	return MXFAssertions{aw.out.Expect(actual, extra...)}
}

type gomegaExpect interface {
	Expect(actual interface{}, extra ...interface{}) types.Assertion
}

type CompleteTest struct {
	segment      *segmentTest
	gomegaExpect Expecter
	// tester       Tester
}

// Expect calls the gomega expect assertion
func (ct CompleteTest) Expect(actual interface{}, extra ...interface{}) Assertions {

	return ct.gomegaExpect.Expect(actual, extra...)
}

// Test interface is the MXF test parameters
type Test interface {
	Tester
	Expecter
	testPass() bool
}

// Expecter is a workaround to wrap the gomega/internal expect object
type Expecter interface {
	Expect(actual interface{}, extra ...interface{}) Assertions
}

// Tester is a workaround to wrap the gomega/internal test object
type Tester interface {
	Test(message string, specDetail SpecDetails, asserts ...bool)
}

// SpecDetails contains the information about the specification
// that made the test. It can be written with %s formatting
type SpecDetails struct {
	DocName, Section, Command string
	CommandCount              int
}

// NewSpec generates a new specDetail struct
func NewSpec(docName, section, command string, commandCount int) SpecDetails {
	// is there a parent required
	return SpecDetails{DocName: docName, Section: section, Command: command, CommandCount: commandCount}
}

// String allows spec details to be written as a shorthand string
func (s SpecDetails) String() string {
	// is there a parent required
	return fmt.Sprintf("%s,%s,%s,%v", s.DocName, s.Section, s.Command, s.CommandCount)
}

// test pass returns if the previous test ran
func (c *CompleteTest) testPass() bool {
	return c.segment.testPass
}

// segmentTest contains all the internal workings for interacting with gomega
type segmentTest struct {

	// use the assertions to compare the error
	// generate a header
	// have an incremental counter of tests
	testCount int
	failCount int

	// segment pass

	// handle the errors when things fail
	errChannel chan string
	// did the test pass or fail
	testPass bool

	testReport TestSection
}

func (s *segmentTest) Helper() {
	// leave as an empty call for the moment
}

// FatalF is run when a test fails
func (s *segmentTest) Fatalf(format string, args ...interface{}) {
	// pipe the gomega err to be handled by the test wrapper
	s.errChannel <- fmt.Sprintf(format, args...)

}

// Assertions wraps the gomega types assertions
// with the additional Shall and ShallNot assertion
type Assertions interface {
	Shall(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool
	ShallNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool
	types.Assertion
}

// MXFAssertions wraps the basic types.assertions with some
// extra names to allow the MXf specification to be
// written as tests.
type MXFAssertions struct {
	standard types.Assertion
}

// Shall wraps the To assertion and behaves in the same way
func (e MXFAssertions) Shall(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}

// ShallNot wraps the ToNot assertion and behaves in the same way
func (e MXFAssertions) ShallNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.ToNot(matcher, optionalDescription...)
}

func (e MXFAssertions) NotTo(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.NotTo(matcher, optionalDescription...)
}
func (e MXFAssertions) To(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e MXFAssertions) ToNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.ToNot(matcher, optionalDescription...)
}
func (e MXFAssertions) Should(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e MXFAssertions) ShouldNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return e.standard.To(matcher, optionalDescription...)
}
func (e MXFAssertions) WithOffset(offset int) types.Assertion {
	return e.standard.WithOffset(offset)
}

func (e MXFAssertions) Error() types.Assertion {
	return e.standard.Error()
}
