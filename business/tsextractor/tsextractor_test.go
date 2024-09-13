package tsextractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeAlignment_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows
	from, to := computeTimeAlignment(3600, 60)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}

func TestTimeAlignment_15minTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows
	from, to := computeTimeAlignment(3600, 15)
	assert.Equal(t, int64(900), to.Unix()-from.Unix())
}

func TestTimeAlignment_15min_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows and 15min resolution
	from, to := computeTimeAlignment(900, 60)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}

func TestTimeAlignment_5min_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows and 5min resolution
	from, to := computeTimeAlignment(300, 60)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}

func TestTimeAlignment_raw_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows and 5min resolution
	from, to := computeTimeAlignment(-1, 60)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}
