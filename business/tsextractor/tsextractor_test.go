package tsextractor

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	iotMocks "github.com/arduino/aws-s3-integration/internal/iot/mocks"
	iotclient "github.com/arduino/iot-client-go/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTimeAlignment_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows, not aligned
	nowTruncated := time.Now().UTC().Truncate(time.Duration(300) * time.Second).Add(-time.Duration(300) * time.Second)
	fromTuncated := nowTruncated.Add(-time.Hour)
	from, to := computeTimeAlignment(300, 60, false)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
	assert.Equal(t, nowTruncated, to)
	assert.Equal(t, fromTuncated, from)
}

func TestTimeAlignment_HourlyTimeWindows_aligned(t *testing.T) {
	// Test the time alignment with hourly time windows, complete last hour
	nowTruncated := time.Now().UTC().Truncate(time.Hour)
	fromTuncated := nowTruncated.Add(-time.Hour)
	from, to := computeTimeAlignment(300, 60, true)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
	assert.Equal(t, nowTruncated, to)
	assert.Equal(t, fromTuncated, from)
}

func TestTimeAlignment_15minTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows
	from, to := computeTimeAlignment(3600, 15, false)
	assert.Equal(t, int64(900), to.Unix()-from.Unix())
}

func TestTimeAlignment_15min_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows and 15min resolution
	from, to := computeTimeAlignment(900, 60, false)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}

func TestTimeAlignment_5min_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows and 5min resolution
	from, to := computeTimeAlignment(300, 60, false)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}

func TestTimeAlignment_raw_HourlyTimeWindows(t *testing.T) {
	// Test the time alignment with hourly time windows and 5min resolution
	from, to := computeTimeAlignment(-1, 60, false)
	assert.Equal(t, int64(3600), to.Unix()-from.Unix())
}

func toPtr(val string) *string {
	return &val
}

func TestExtractionFlow_defaultAggregation(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	thingId := "91f30213-2bd7-480a-b1dc-f31b01840e7e"
	propertyId := "c86f4ed9-7f52-4bd3-bdc6-b2936bec68ac"

	// Init client
	iotcl := iotMocks.NewAPI(t)

	now := time.Now()
	responses := []iotclient.ArduinoSeriesResponse{
		{
			Aggregation: toPtr("AVG"),
			Query:       fmt.Sprintf("property.%s", propertyId),
			Times:       []time.Time{now.Add(-time.Minute * 1), now},
			Values:      []float64{1.0, 2.0},
			CountValues: 2,
		},
	}
	samples := iotclient.ArduinoSeriesBatch{
		Responses: responses,
	}
	iotcl.On("GetTimeSeriesByThing", ctx, thingId, mock.Anything, mock.Anything, int64(300), "AVG").Return(&samples, false, nil)

	tsextractorClient := New(iotcl, logger)

	propCount := int64(1)
	thingsMap := make(map[string]iotclient.ArduinoThing)
	thingsMap[thingId] = iotclient.ArduinoThing{
		Id:   thingId,
		Name: "test",
		Properties: []iotclient.ArduinoProperty{
			{
				Name: "ptest",
				Id:   propertyId,
				Type: "FLOAT",
			},
		},
		PropertiesCount: &propCount,
	}

	writer, from, err := tsextractorClient.ExportTSToFile(ctx, 60, thingsMap, 300, "AVG", false)
	assert.NoError(t, err)
	assert.NotNil(t, writer)
	assert.NotNil(t, from)

	writer.Close()
	defer writer.Delete()

	outF, err := os.Open(writer.GetFilePath())
	assert.NoError(t, err)
	defer outF.Close()
	content, err := io.ReadAll(outF)
	assert.NoError(t, err)
	entries := []string{
		"timestamp,thing_id,thing_name,property_id,property_name,property_type,value",
		"91f30213-2bd7-480a-b1dc-f31b01840e7e,test,c86f4ed9-7f52-4bd3-bdc6-b2936bec68ac,ptest,FLOAT,1,AVG",
		"91f30213-2bd7-480a-b1dc-f31b01840e7e,test,c86f4ed9-7f52-4bd3-bdc6-b2936bec68ac,ptest,FLOAT,2,AVG",
	}
	for _, entry := range entries {
		assert.Contains(t, string(content), entry)
	}
	fmt.Println(string(content))
}
