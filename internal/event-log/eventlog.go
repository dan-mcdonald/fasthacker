package eventlog

// Interface for reading and writing events to a CSV log
// The CSV log has the following format:
// unix seconds, unix nanoseconds, event type, event data (JSON)

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

type EventType string

const (
	TypeItem = "item"
	TypeUser = "user"
)

type Event struct {
	RxTime    time.Time
	EventType EventType
	Data      []byte
}

type EventLog struct {
	file      *os.File
	csvReader *csv.Reader
	csvWriter *csv.Writer
	fullyRead bool
}

func (e *EventLog) Close() error {
	return e.file.Close()
}

func NewEventLog(path string) (*EventLog, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 4
	reader.ReuseRecord = true

	return &EventLog{
		file:      file,
		csvReader: reader,
		csvWriter: csv.NewWriter(file),
		fullyRead: false,
	}, nil
}

func (e *EventLog) Read() (Event, error) {
	record, err := e.csvReader.Read()
	if err != nil {
		return Event{}, err
	}
	unixSecs, err := strconv.ParseInt(record[0], 10, 64)
	if err != nil {
		return Event{}, fmt.Errorf("eventlog.Read: error parsing first column (unix seconds): %w", err)
	}
	unixNs, err := strconv.ParseInt(record[1], 10, 64)
	if err != nil {
		return Event{}, fmt.Errorf("eventlog.Read: error parsing second column (unix nanoseconds): %w", err)
	}
	return Event{
		RxTime:    time.Unix(unixSecs, unixNs),
		EventType: EventType(record[2]),
		Data:      []byte(record[3]),
	}, nil
}
