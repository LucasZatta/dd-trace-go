// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package gotesting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/constants"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/utils/net"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

var currentM *testing.M
var mTracer mocktracer.Tracer

// TestMain is the entry point for testing and runs before any test.
func TestMain(m *testing.M) {
	log.SetLevel(log.LevelDebug)

	// We need to spawn separated test process for each scenario
	scenarios := []string{"TestFlakyTestRetries", "TestEarlyFlakeDetection", "TestFlakyTestRetriesAndEarlyFlakeDetection", "TestIntelligentTestRunner"}

	if internal.BoolEnv(scenarios[0], false) {
		fmt.Printf("Scenario %s started.\n", scenarios[0])
		runFlakyTestRetriesTests(m)
	} else if internal.BoolEnv(scenarios[1], false) {
		fmt.Printf("Scenario %s started.\n", scenarios[1])
		runEarlyFlakyTestDetectionTests(m)
	} else if internal.BoolEnv(scenarios[2], false) {
		fmt.Printf("Scenario %s started.\n", scenarios[2])
		runFlakyTestRetriesWithEarlyFlakyTestDetectionTests(m)
	} else if internal.BoolEnv(scenarios[3], false) {
		fmt.Printf("Scenario %s started.\n", scenarios[3])
		runIntelligentTestRunnerTests(m)
	} else {
		fmt.Println("Starting tests...")
		for _, v := range scenarios {
			cmd := exec.Command(os.Args[0], os.Args[1:]...)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=true", v))
			fmt.Printf("Running scenario: %s:\n", v)
			err := cmd.Run()
			fmt.Printf("Done.\n\n")
			if err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					fmt.Printf("Scenario %s failed with exit code: %d\n", v, exiterr.ExitCode())
					os.Exit(exiterr.ExitCode())
				} else {
					fmt.Printf("cmd.Run: %v\n", err)
					os.Exit(1)
				}
				break
			}
		}
	}

	os.Exit(0)
}

func runFlakyTestRetriesTests(m *testing.M) {
	// mock the settings api to enable automatic test retries
	server := setUpHttpServer(true, false, nil, false, nil)
	defer server.Close()

	// set a custom retry count
	os.Setenv(constants.CIVisibilityFlakyRetryCountEnvironmentVariable, "10")

	// initialize the mock tracer for doing assertions on the finished spans
	currentM = m
	mTracer = integrations.InitializeCIVisibilityMock()

	// execute the tests, we are expecting some tests to fail and check the assertion later
	exitCode := RunM(m)
	if exitCode != 0 {
		panic("expected the exit code to be 0. Got exit code: " + fmt.Sprintf("%d", exitCode))
	}

	// get all finished spans
	finishedSpans := mTracer.FinishedSpans()

	// 1 session span
	// 1 module span
	// 2 suite span (testing_test.go and reflections_test.go)
	// 5 tests from reflections_test.go
	// 1 TestMyTest01
	// 1 TestMyTest02 + 2 subtests
	// 1 Test_Foo + 3 subtests
	// 1 TestSkip
	// 1 TestRetryWithPanic + 3 retry tests from testing_test.go
	// 1 TestRetryWithFail + 3 retry tests from testing_test.go
	// 1 TestNormalPassingAfterRetryAlwaysFail
	// 1 TestEarlyFlakeDetection

	// check spans by resource name
	checkSpansByResourceName(finishedSpans, "gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting", 1)
	checkSpansByResourceName(finishedSpans, "reflections_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest01", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01/sub03", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/yellow_should_return_color", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/banana_should_return_fruit", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/duck_should_return_animal", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestSkip", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithPanic", 4)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithFail", 4)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestNormalPassingAfterRetryAlwaysFail", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestEarlyFlakeDetection", 1)

	// check spans by tag
	checkSpansByTagName(finishedSpans, constants.TestIsRetry, 6)

	// check spans by type
	checkSpansByType(finishedSpans,
		28,
		1,
		1,
		2,
		24,
		0)

	fmt.Println("All tests passed.")
	os.Exit(0)
}

func runEarlyFlakyTestDetectionTests(m *testing.M) {
	// mock the settings api to enable automatic test retries
	server := setUpHttpServer(false, true, &net.EfdResponseData{
		Tests: net.EfdResponseDataModules{
			"gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting": net.EfdResponseDataSuites{
				"reflections_test.go": []string{
					"TestGetFieldPointerFrom",
					"TestGetInternalTestArray",
					"TestGetInternalBenchmarkArray",
					"TestCommonPrivateFields_AddLevel",
					"TestGetBenchmarkPrivateFields",
				},
			},
		},
	}, false, nil)
	defer server.Close()

	// initialize the mock tracer for doing assertions on the finished spans
	currentM = m
	mTracer = integrations.InitializeCIVisibilityMock()

	// execute the tests, we are expecting some tests to fail and check the assertion later
	exitCode := RunM(m)
	if exitCode != 0 {
		panic("expected the exit code to be 0. Got exit code: " + fmt.Sprintf("%d", exitCode))
	}

	// get all finished spans
	finishedSpans := mTracer.FinishedSpans()

	// 1 session span
	// 1 module span
	// 2 suite span (testing_test.go and reflections_test.go)
	// 5 tests from reflections_test.go
	// 11 TestMyTest01
	// 11 TestMyTest02 + 22 subtests
	// 11 Test_Foo + 33 subtests
	// 11 TestSkip
	// 11 TestRetryWithPanic
	// 11 TestRetryWithFail
	// 11 TestNormalPassingAfterRetryAlwaysFail
	// 11 TestEarlyFlakeDetection
	// 22 normal spans from testing_test.go

	// check spans by resource name
	checkSpansByResourceName(finishedSpans, "gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting", 1)
	checkSpansByResourceName(finishedSpans, "reflections_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest01", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01/sub03", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/yellow_should_return_color", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/banana_should_return_fruit", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/duck_should_return_animal", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestSkip", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithPanic", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithFail", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestNormalPassingAfterRetryAlwaysFail", 11)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestEarlyFlakeDetection", 11)

	// check spans by tag
	checkSpansByTagName(finishedSpans, constants.TestIsNew, 143)
	checkSpansByTagName(finishedSpans, constants.TestIsRetry, 130)

	// check spans by type
	checkSpansByType(finishedSpans,
		152,
		1,
		1,
		2,
		148,
		0)

	fmt.Println("All tests passed.")
	os.Exit(0)
}

func runFlakyTestRetriesWithEarlyFlakyTestDetectionTests(m *testing.M) {
	// mock the settings api to enable automatic test retries
	server := setUpHttpServer(true, true, &net.EfdResponseData{
		Tests: net.EfdResponseDataModules{
			"gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting": net.EfdResponseDataSuites{
				"reflections_test.go": []string{
					"TestGetFieldPointerFrom",
					"TestGetInternalTestArray",
					"TestGetInternalBenchmarkArray",
					"TestCommonPrivateFields_AddLevel",
					"TestGetBenchmarkPrivateFields",
				},
				"testing_test.go": []string{
					"TestMyTest01",
					"TestMyTest02",
					"Test_Foo",
					"TestWithExternalCalls",
					"TestSkip",
					"TestRetryWithPanic",
					"TestRetryWithFail",
					"TestRetryAlwaysFail",
					"TestNormalPassingAfterRetryAlwaysFail",
				},
			},
		},
	}, false, nil)
	defer server.Close()

	// set a custom retry count
	os.Setenv(constants.CIVisibilityFlakyRetryCountEnvironmentVariable, "10")

	// initialize the mock tracer for doing assertions on the finished spans
	currentM = m
	mTracer = integrations.InitializeCIVisibilityMock()

	// execute the tests, we are expecting some tests to fail and check the assertion later
	exitCode := RunM(m)
	if exitCode != 0 {
		panic("expected the exit code to be 0. Got exit code: " + fmt.Sprintf("%d", exitCode))
	}

	// get all finished spans
	finishedSpans := mTracer.FinishedSpans()

	// 1 session span
	// 1 module span
	// 2 suite span (testing_test.go and reflections_test.go)
	// 5 tests from reflections_test.go
	// 1 TestMyTest01
	// 1 TestMyTest02 + 2 subtests
	// 1 Test_Foo + 3 subtests
	// 1 TestWithExternalCalls + 2 subtests
	// 1 TestSkip
	// 1 TestRetryWithPanic + 3 retry tests from testing_test.go
	// 1 TestRetryWithFail + 3 retry tests from testing_test.go
	// 1 TestNormalPassingAfterRetryAlwaysFail
	// 1 TestEarlyFlakeDetection + 10 EFD retries
	// 2 normal spans from testing_test.go

	// check spans by resource name
	checkSpansByResourceName(finishedSpans, "gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting", 1)
	checkSpansByResourceName(finishedSpans, "reflections_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest01", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01/sub03", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/yellow_should_return_color", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/banana_should_return_fruit", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/duck_should_return_animal", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestSkip", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithPanic", 4)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithFail", 4)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestNormalPassingAfterRetryAlwaysFail", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestEarlyFlakeDetection", 11)

	// check spans by tag
	checkSpansByTagName(finishedSpans, constants.TestIsNew, 11)
	checkSpansByTagName(finishedSpans, constants.TestIsRetry, 16)

	// check spans by type
	checkSpansByType(finishedSpans,
		38,
		1,
		1,
		2,
		34,
		0)

	fmt.Println("All tests passed.")
	os.Exit(0)
}

func runIntelligentTestRunnerTests(m *testing.M) {
	// mock the settings api to enable automatic test retries
	server := setUpHttpServer(true, false, nil, true, []net.SkippableResponseDataAttributes{
		{
			Suite: "testing_test.go",
			Name:  "TestMyTest01",
		},
		{
			Suite: "testing_test.go",
			Name:  "TestMyTest02",
		},
		{
			Suite: "testing_test.go",
			Name:  "Test_Foo",
		},
		{
			Suite: "testing_test.go",
			Name:  "TestRetryWithPanic",
		},
		{
			Suite: "testing_test.go",
			Name:  "TestRetryWithFail",
		},
		{
			Suite: "testing_test.go",
			Name:  "TestRetryAlwaysFail",
		},
	})
	defer server.Close()

	// initialize the mock tracer for doing assertions on the finished spans
	currentM = m
	mTracer = integrations.InitializeCIVisibilityMock()

	// execute the tests, we are expecting some tests to fail and check the assertion later
	exitCode := RunM(m)
	if exitCode != 0 {
		panic("expected the exit code to be 0. All tests should pass (failed ones should be skipped by ITR).")
	}

	// get all finished spans
	finishedSpans := mTracer.FinishedSpans()

	// 1 session span
	// 1 module span
	// 2 suite span (testing_test.go and reflections_test.go)
	// 5 tests from reflections_test.go
	// 1 TestMyTest01
	// 1 TestMyTest02
	// 1 Test_Foo
	// 1 TestSkip
	// 1 TestRetryWithPanic
	// 1 TestRetryWithFail
	// 1 TestRetryAlwaysFail
	// 1 TestNormalPassingAfterRetryAlwaysFail
	// 1 TestEarlyFlakeDetection

	// check spans by resource name
	checkSpansByResourceName(finishedSpans, "gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting", 1)
	checkSpansByResourceName(finishedSpans, "reflections_test.go", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go", 1)
	itrTest01 := checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest01", 1)
	itrTest02 := checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01", 0)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestMyTest02/sub01/sub03", 0)
	itrTest03 := checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/yellow_should_return_color", 0)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/banana_should_return_fruit", 0)
	checkSpansByResourceName(finishedSpans, "testing_test.go.Test_Foo/duck_should_return_animal", 0)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestSkip", 1)
	itrTest04 := checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithPanic", 1)
	itrTest05 := checkSpansByResourceName(finishedSpans, "testing_test.go.TestRetryWithFail", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestNormalPassingAfterRetryAlwaysFail", 1)
	checkSpansByResourceName(finishedSpans, "testing_test.go.TestEarlyFlakeDetection", 1)

	// check ITR spans
	var itrTests []mocktracer.Span
	itrTests = append(itrTests, itrTest01...)
	itrTests = append(itrTests, itrTest02...)
	itrTests = append(itrTests, itrTest03...)
	itrTests = append(itrTests, itrTest04...)
	itrTests = append(itrTests, itrTest05...)
	checkSpansByTagValue(itrTests, constants.TestStatus, constants.TestStatusSkip, 5)
	checkSpansByTagValue(itrTests, constants.TestSkipReason, constants.SkippedByITRReason, 5)

	// check spans by type
	checkSpansByType(finishedSpans,
		17,
		1,
		1,
		2,
		13,
		0)

	fmt.Println("All tests passed.")
	os.Exit(0)
}

func checkSpansByType(finishedSpans []mocktracer.Span,
	totalFinishedSpansCount int, sessionSpansCount int, moduleSpansCount int,
	suiteSpansCount int, testSpansCount int, normalSpansCount int) {
	calculatedFinishedSpans := len(finishedSpans)
	fmt.Printf("Number of spans received: %d\n", calculatedFinishedSpans)
	if calculatedFinishedSpans < totalFinishedSpansCount {
		panic(fmt.Sprintf("expected at least %d finished spans, got %d", totalFinishedSpansCount, calculatedFinishedSpans))
	}

	sessionSpans := getSpansWithType(finishedSpans, constants.SpanTypeTestSession)
	calculatedSessionSpans := len(sessionSpans)
	fmt.Printf("Number of sessions received: %d\n", calculatedSessionSpans)
	showResourcesNameFromSpans(sessionSpans)
	if calculatedSessionSpans != sessionSpansCount {
		panic(fmt.Sprintf("expected exactly %d session span, got %d", sessionSpansCount, calculatedSessionSpans))
	}

	moduleSpans := getSpansWithType(finishedSpans, constants.SpanTypeTestModule)
	calculatedModuleSpans := len(moduleSpans)
	fmt.Printf("Number of modules received: %d\n", calculatedModuleSpans)
	showResourcesNameFromSpans(moduleSpans)
	if calculatedModuleSpans != moduleSpansCount {
		panic(fmt.Sprintf("expected exactly %d module span, got %d", moduleSpansCount, calculatedModuleSpans))
	}

	suiteSpans := getSpansWithType(finishedSpans, constants.SpanTypeTestSuite)
	calculatedSuiteSpans := len(suiteSpans)
	fmt.Printf("Number of suites received: %d\n", calculatedSuiteSpans)
	showResourcesNameFromSpans(suiteSpans)
	if calculatedSuiteSpans != suiteSpansCount {
		panic(fmt.Sprintf("expected exactly %d suite spans, got %d", suiteSpansCount, calculatedSuiteSpans))
	}

	testSpans := getSpansWithType(finishedSpans, constants.SpanTypeTest)
	calculatedTestSpans := len(testSpans)
	fmt.Printf("Number of tests received: %d\n", calculatedTestSpans)
	showResourcesNameFromSpans(testSpans)
	if calculatedTestSpans != testSpansCount {
		panic(fmt.Sprintf("expected exactly %d test spans, got %d", testSpansCount, calculatedTestSpans))
	}

	normalSpans := getSpansWithType(finishedSpans, ext.SpanTypeHTTP)
	calculatedNormalSpans := len(normalSpans)
	fmt.Printf("Number of http spans received: %d\n", calculatedNormalSpans)
	showResourcesNameFromSpans(normalSpans)
	if calculatedNormalSpans != normalSpansCount {
		panic(fmt.Sprintf("expected exactly %d normal spans, got %d", normalSpansCount, calculatedNormalSpans))
	}
}

func checkSpansByResourceName(finishedSpans []mocktracer.Span, resourceName string, count int) []mocktracer.Span {
	spans := getSpansWithResourceName(finishedSpans, resourceName)
	numOfSpans := len(spans)
	if numOfSpans != count {
		panic(fmt.Sprintf("expected exactly %d spans with resource name: %s, got %d", count, resourceName, numOfSpans))
	}

	return spans
}

func checkSpansByTagName(finishedSpans []mocktracer.Span, tagName string, count int) []mocktracer.Span {
	spans := getSpansWithTagName(finishedSpans, tagName)
	numOfSpans := len(spans)
	if numOfSpans != count {
		panic(fmt.Sprintf("expected exactly %d spans with tag name: %s, got %d", count, tagName, numOfSpans))
	}

	return spans
}

func checkSpansByTagValue(finishedSpans []mocktracer.Span, tagName, tagValue string, count int) []mocktracer.Span {
	spans := getSpansWithTagNameAndValue(finishedSpans, tagName, tagValue)
	numOfSpans := len(spans)
	if numOfSpans != count {
		panic(fmt.Sprintf("expected exactly %d spans with tag name: %s and value %s, got %d", count, tagName, tagValue, numOfSpans))
	}

	return spans
}

type (
	skippableResponse struct {
		Meta skippableResponseMeta   `json:"meta"`
		Data []skippableResponseData `json:"data"`
	}

	skippableResponseMeta struct {
		CorrelationID string `json:"correlation_id"`
	}

	skippableResponseData struct {
		ID         string                              `json:"id"`
		Type       string                              `json:"type"`
		Attributes net.SkippableResponseDataAttributes `json:"attributes"`
	}
)

func setUpHttpServer(flakyRetriesEnabled bool,
	earlyFlakyDetectionEnabled bool, earlyFlakyDetectionData *net.EfdResponseData,
	itrEnabled bool, itrData []net.SkippableResponseDataAttributes) *httptest.Server {
	// mock the settings api to enable automatic test retries
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("MockApi received request: %s\n", r.URL.Path)

		// Settings request
		if r.URL.Path == "/api/v2/libraries/tests/services/setting" {
			w.Header().Set("Content-Type", "application/json")
			response := struct {
				Data struct {
					ID         string                   `json:"id"`
					Type       string                   `json:"type"`
					Attributes net.SettingsResponseData `json:"attributes"`
				} `json:"data,omitempty"`
			}{}

			// let's enable flaky test retries
			response.Data.Attributes = net.SettingsResponseData{
				FlakyTestRetriesEnabled: flakyRetriesEnabled,
				ItrEnabled:              itrEnabled,
				TestsSkipping:           itrEnabled,
			}
			response.Data.Attributes.EarlyFlakeDetection.Enabled = earlyFlakyDetectionEnabled
			response.Data.Attributes.EarlyFlakeDetection.SlowTestRetries.FiveS = 10
			response.Data.Attributes.EarlyFlakeDetection.SlowTestRetries.TenS = 5
			response.Data.Attributes.EarlyFlakeDetection.SlowTestRetries.ThirtyS = 3
			response.Data.Attributes.EarlyFlakeDetection.SlowTestRetries.FiveM = 2

			fmt.Printf("MockApi sending response: %v\n", response)
			json.NewEncoder(w).Encode(&response)
		} else if earlyFlakyDetectionEnabled && r.URL.Path == "/api/v2/ci/libraries/tests" {
			w.Header().Set("Content-Type", "application/json")
			response := struct {
				Data struct {
					ID         string              `json:"id"`
					Type       string              `json:"type"`
					Attributes net.EfdResponseData `json:"attributes"`
				} `json:"data,omitempty"`
			}{}

			if earlyFlakyDetectionData != nil {
				response.Data.Attributes = *earlyFlakyDetectionData
			}

			fmt.Printf("MockApi sending response: %v\n", response)
			json.NewEncoder(w).Encode(&response)
		} else if r.URL.Path == "/api/v2/git/repository/search_commits" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
		} else if r.URL.Path == "/api/v2/git/repository/packfile" {
			w.WriteHeader(http.StatusAccepted)
		} else if itrEnabled && r.URL.Path == "/api/v2/ci/tests/skippable" {
			w.Header().Set("Content-Type", "application/json")
			response := skippableResponse{
				Meta: skippableResponseMeta{
					CorrelationID: "correlation_id",
				},
				Data: []skippableResponseData{},
			}
			for i, data := range itrData {
				response.Data = append(response.Data, skippableResponseData{
					ID:         fmt.Sprintf("id_%d", i),
					Type:       "type",
					Attributes: data,
				})
			}
			fmt.Printf("MockApi sending response: %v\n", response)
			json.NewEncoder(w).Encode(&response)
		} else {
			http.NotFound(w, r)
		}
	}))

	// set the custom agentless url and the flaky retry count env-var
	fmt.Printf("Using mockapi at: %s\n", server.URL)
	os.Setenv(constants.CIVisibilityAgentlessEnabledEnvironmentVariable, "1")
	os.Setenv(constants.CIVisibilityAgentlessURLEnvironmentVariable, server.URL)
	os.Setenv(constants.APIKeyEnvironmentVariable, "12345")

	return server
}

func getSpansWithType(spans []mocktracer.Span, spanType string) []mocktracer.Span {
	var result []mocktracer.Span
	for _, span := range spans {
		if span.Tag(ext.SpanType) == spanType {
			result = append(result, span)
		}
	}

	return result
}

func getSpansWithResourceName(spans []mocktracer.Span, resourceName string) []mocktracer.Span {
	var result []mocktracer.Span
	for _, span := range spans {
		if span.Tag(ext.ResourceName) == resourceName {
			result = append(result, span)
		}
	}

	return result
}

func getSpansWithTagName(spans []mocktracer.Span, tag string) []mocktracer.Span {
	var result []mocktracer.Span
	for _, span := range spans {
		if span.Tag(tag) != nil {
			result = append(result, span)
		}
	}

	return result
}

func getSpansWithTagNameAndValue(spans []mocktracer.Span, tag, value string) []mocktracer.Span {
	var result []mocktracer.Span
	for _, span := range spans {
		if span.Tag(tag) == value {
			result = append(result, span)
		}
	}

	return result
}

func showResourcesNameFromSpans(spans []mocktracer.Span) {
	for i, span := range spans {
		fmt.Printf("  [%d] = %v\n", i, span.Tag(ext.ResourceName))
	}
}
