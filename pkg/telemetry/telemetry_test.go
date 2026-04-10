package telemetry

import "testing"

func TestNormalizeSamplingRatio(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{name: "negative", input: -1, want: 0},
		{name: "zero", input: 0, want: 0},
		{name: "mid", input: 0.25, want: 0.25},
		{name: "one", input: 1, want: 1},
		{name: "large", input: 5, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSamplingRatio(tt.input); got != tt.want {
				t.Fatalf("normalizeSamplingRatio(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
