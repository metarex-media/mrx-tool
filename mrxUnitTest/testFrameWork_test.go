package mrxUnitTest

import (
	"fmt"
	"io"
	"testing"

	"github.com/onsi/gomega"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHandler(t *testing.T) {

	testFail := func(t Test) bool { return t.Expect(nil).ShallNot(gomega.BeNil(), "Uh oh your test failed") }
	testPass := func(t Test) bool { return t.Expect(nil).Shall(gomega.BeNil()) }

	tests := []func(t Test) bool{testFail, testPass}
	expected := []bool{false, true}
	errMess := []string{"Uh oh your test failed\nExpected\n    <nil>: nil\nnot to be nil", ""}

	for i, test := range tests {
		tc := NewTestContext(io.Discard)

		var testPassRes bool
		tc.Header("some message", func(t Test) {
			t.Test("", SpecDetails{}, test(t))
			// does the test pass correctly return the result?
			testPassRes = t.testPass()
		})

		Convey("Checking that the outcomes of the test are logged in the report", t, func() {
			Convey(fmt.Sprintf("running a test with an expected result of %v", expected[i]), func() {
				Convey("The test matches the result and the associated functions and values match as well", func() {
					// check the one test was run
					So(len(tc.report.Tests), ShouldResemble, 1)
					So(tc.report.Tests[0].Pass, ShouldResemble, expected[i])
					// check teh handler also works
					So(testPassRes, ShouldResemble, expected[i])
					// check the message exists
					So(tc.report.Tests[0].Tests[0].Checks[0].ErrMessage, ShouldResemble, errMess[i])
				})
			})
		})
	}

	// ensure the added tests are actually added
	tc := NewTestContext(io.Discard)
	skippedTests := []skippedTest{
		{TestKey: "Test Key for a partition", Desc: "partition test"},
		{TestKey: "Test Key for a node of some sort", Desc: "node test"},
		{TestKey: "A bonus test key", Desc: "bonus test"},
	}

	for _, st := range skippedTests {
		tc.RegisterSkippedTest(st.TestKey, st.Desc)
	}

	Convey("Checking that the outcomes of the test are logged in the report", t, func() {
		Convey(fmt.Sprintf("running a test with an expected result of %v", "expected[i]"), func() {
			Convey("The test matches the result and the associated functions and values match as well", func() {
				// check the one test was run
				So(tc.report.SkippedTests, ShouldResemble, skippedTests)

			})
		})
	})
	// tc.RegisterSkippedTest()
	/*

		run the header then handle the test functions and make sure they call correctly. - write directly to the buffer so we can process the result
		check anything else

	*/

}
