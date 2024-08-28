package mrxUnitTest

import (
	"fmt"
	"io"
	"math"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
	"gopkg.in/yaml.v3"
)

// DrawGraph takes the yaml log generated from the
/*
MRXTest function
and draws a graphical representation of it.

It is a bar chart with the total pass and failure counts
as the vertical two bars.
*/
func DrawGraph(reportStream io.Reader, dest io.Writer) error {

	reportBytes, err := io.ReadAll(reportStream)

	if err != nil {
		return fmt.Errorf("error extracting report bytes : %v", err)
	}

	var report Report
	err = yaml.Unmarshal(reportBytes, &report)

	if err != nil {
		return fmt.Errorf("error extracting the report from the yaml file %v", err)
	}

	// count the results of the test
	passCount := 0
	failCount := 0

	for _, t := range report.Tests {
		passCount += t.PassCount
		failCount += t.FailCount
	}

	max := passCount
	if passCount < failCount {
		max = failCount
	}
	max += max / 6
	// want a max y axis height of 5
	if max < 5 {
		max = 5
	}

	// generate the Y axis chart ticks
	ticks := make([]chart.Tick, 6)
	step := int(math.RoundToEven(float64(max) / 5))
	for i := 0; i < 6; i++ {
		ticks[i] = chart.Tick{Value: float64(i * step), Label: fmt.Sprintf("%v", i*step)}
	}

	graph := chart.BarChart{
		Title: fmt.Sprintf("MXF Report results : %v", report.TestPass),
		YAxis: chart.YAxis{
			Ticks: ticks, Range: &chart.ContinuousRange{Min: 0, Max: float64(max)}},

		Width:  500,
		Height: 300,
	}

	red := drawing.Color{R: 0xff, A: 0xff}
	green := drawing.Color{G: 0xff, A: 0xff}
	colours := []drawing.Color{red, green}
	empty := drawing.Color{R: 0xff, G: 0xff, A: 0x00}
	values := make([]chart.Value, 2)

	counts := []float64{float64(failCount), float64(passCount)}
	fields := []string{"fail", "pass"}

	for i, c := range counts {

		if c == 0 {
			values[i] = chart.Value{
				Label: fields[i], Value: c,
				Style: chart.Style{FillColor: empty, StrokeColor: empty},
			}
		} else {
			values[i] = chart.Value{
				Label: fields[i], Value: c,
				Style: chart.Style{FillColor: colours[i], StrokeColor: colours[i]},
			}
		}
	}

	graph.Bars = values

	// render the graph
	return graph.Render(chart.PNG, dest)

}
