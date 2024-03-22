package reporters

import (
	"flag"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/ginkgo/types"
)

var (
	// reportDir is used to set the output directory for JUnit artifacts
	reportDir string
)

func init() {
	flag.StringVar(&reportDir, "report-dir", "", "Set report directory for artifact output")
}

type NewlineReporter struct{}

func (reporter *NewlineReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	fmt.Printf("Test: %s - %+v\n\n", specSummary.ComponentTexts[1], specSummary.State)
}

// Implement other methods of the ginkgo.Reporter interface with empty bodies, as they are required but may not be needed.
func (reporter *NewlineReporter) SpecSuiteWillBegin(config.GinkgoConfigType, *types.SuiteSummary) {}
func (reporter *NewlineReporter) BeforeSuiteDidRun(*types.SetupSummary)                           {}
func (reporter *NewlineReporter) AfterSuiteDidRun(*types.SetupSummary)                            {}
func (reporter *NewlineReporter) SpecWillRun(*types.SpecSummary)                                  {}
func (reporter *NewlineReporter) SpecSuiteDidEnd(*types.SuiteSummary)                             {}

// Reporters creates the ginkgo reporters for the test suites
func Reporters() []ginkgo.Reporter {
	now, _ := time.Now().MarshalText()
	reps := []ginkgo.Reporter{&NewlineReporter{}} // Use the custom NewlineReporter
	if reportDir != "" {
		reps = append(reps, reporters.NewJUnitReporter(fmt.Sprintf("%s/junit_%s_%d.xml", reportDir, string(now), config.GinkgoConfig.ParallelNode)))
	}
	return reps
}
