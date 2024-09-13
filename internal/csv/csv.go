package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	baseTmpStorage = "/tmp"
)

var csvHeader = []string{"timestamp", "thing_id", "thing_name", "property_id", "property_name", "property_type", "value"}

func NewWriter(destinationHour time.Time, logger *logrus.Entry) (*CsvWriter, error) {
	filePath := fmt.Sprintf("%s/%s.csv", baseTmpStorage, destinationHour.Format("2006-01-02-15-04"))
	file, err := os.Create(filePath)
	if err != nil {
		logger.Fatalf("failed creating file: %s", err)
	}
	writer := csv.NewWriter(file)
	if err := writer.Write(csvHeader); err != nil {
		logger.Fatalf("failed writing record to file: %s", err)
	}
	return &CsvWriter{
		outFile:   file,
		logger:    logger,
		csvWriter: writer,
		filePath:  filePath,
	}, nil
}

type CsvWriter struct {
	fileWriteLock sync.Mutex
	outFile       *os.File
	logger        *logrus.Entry
	csvWriter     *csv.Writer
	filePath      string
}

func (c *CsvWriter) Write(records [][]string) error {
	c.fileWriteLock.Lock()
	defer c.fileWriteLock.Unlock()

	// Write records to csv file
	for _, record := range records {
		if err := c.csvWriter.Write(record); err != nil {
			return err
		}
	}
	c.csvWriter.Flush()
	return nil
}

func (c *CsvWriter) GetFilePath() string {
	return c.filePath
}

func (c *CsvWriter) Close() error {
	c.logger.Infoln("Closing ouput csv file ", c.outFile.Name())
	if c.csvWriter != nil && c.outFile != nil {
		c.csvWriter.Flush()
		err := c.outFile.Close()
		c.csvWriter = nil
		c.outFile = nil
		return err
	} else {
		return errors.New("no file to close")
	}
}

func (c *CsvWriter) Delete() error {
	if c.outFile != nil {
		c.Close()
	}
	return os.Remove(c.filePath)
}
