package main

import "testing"

func TestSelectAutoPNGScale(t *testing.T) {
	tests := []struct {
		name          string
		requested     float64
		logicalWidth  float64
		logicalHeight float64
		want          float64
	}{
		{name: "small default", logicalWidth: 800, logicalHeight: 600, want: 4},
		{name: "boundary", logicalWidth: 1024, logicalHeight: 1024, want: 4},
		{name: "width exceeds limit", logicalWidth: 1025, logicalHeight: 600, want: 2},
		{name: "height exceeds limit", logicalWidth: 800, logicalHeight: 1025, want: 2},
		{name: "explicit scale wins", requested: 4, logicalWidth: 2000, logicalHeight: 2000, want: 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := selectAutoPNGScale(tc.requested, tc.logicalWidth, tc.logicalHeight)
			if got != tc.want {
				t.Fatalf("got scale %g, want %g", got, tc.want)
			}
		})
	}
}
