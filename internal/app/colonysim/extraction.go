package colonysim

import (
	"math"
	"time"
)

// extractorOutput returns the amount extracted by the cycle of an extractor program ending at runTime.
func extractorOutput(baseValue int64, installTime, runTime time.Time, cycleTime time.Duration) int64 {
	const sec = 10_000_000 // ticks per second
	const decayFactor = 0.012
	const noiseFactor = 0.8
	cycle := int64(cycleTime/time.Second) * sec
	if cycle <= 0 {
		return 0
	}
	timeDiff := (runTime.Unix() - installTime.Unix()) * sec
	cycleNum := max((timeDiff+sec)/cycle-1, 0)
	barWidth := float64(cycle/sec) / 900.0
	t := (float64(cycleNum) + 0.5) * barWidth
	decayValue := float64(baseValue) / (1 + t*decayFactor)
	phaseShift := math.Pow(float64(baseValue), 0.7)
	sinA := math.Cos(phaseShift + t*(1.0/12.0))
	sinB := math.Cos(phaseShift/2.0 + t*(1.0/5.0))
	sinC := math.Cos(t * (1.0 / 2.0))
	sinStuff := max(0, (sinA+sinB+sinC)/3.0)
	barHeight := decayValue * (1 + noiseFactor*sinStuff)
	output := barWidth * barHeight
	// integers are also rounded down, e.g. 123.0 -> 122
	if output == math.Trunc(output) {
		return int64(output) - 1
	}
	return int64(output)
}
