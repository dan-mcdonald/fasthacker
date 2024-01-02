package eventlog

// Interface for reading and writing events to a CSV log
// The CSV log has the following format:
// unix_nanoseconds, event_type, data
// The CSV log is append-only
// The CSV log is not thread-safe

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
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
	e.csvWriter.Flush()
	if err := e.csvWriter.Error(); err != nil {
		e.file.Close()
		return err
	}
	return e.file.Close()
}

const (
	fieldTimeNs    = iota
	fieldEventType = iota
	fieldData      = iota
	fieldEmpty     = iota
	fieldCount     = iota
)

func NewEventLog(path string) (*EventLog, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = fieldCount
	reader.ReuseRecord = true

	return &EventLog{
		file:      file,
		csvReader: reader,
		csvWriter: csv.NewWriter(file),
		fullyRead: false,
	}, nil
}

// Read an event from the log
func (e *EventLog) Read() (Event, error) {
	if e.fullyRead {
		return Event{}, errors.New("eventlog.Read: log already fully read")
	}
	record, err := e.csvReader.Read()
	if err != nil {
		if err == io.EOF {
			e.fullyRead = true
		}
		return Event{}, err
	}
	unixNs, err := strconv.ParseInt(record[fieldTimeNs], 10, 64)
	if err != nil {
		return Event{}, fmt.Errorf("eventlog.Read: error parsing value '%s' as unix nanoseconds: %w", record[fieldTimeNs], err)
	}
	return Event{
		RxTime:    time.Unix(0, unixNs),
		EventType: EventType(record[fieldEventType]),
		Data:      []byte(record[fieldData]),
	}, nil
}

func (e *EventLog) FullyRead() bool {
	return e.fullyRead
}

// Write an event to the log
func (e *EventLog) Write(event Event) error {
	if !e.fullyRead {
		return errors.New("eventlog.Write: log not fully read")
	}
	record := make([]string, fieldCount)
	record[fieldTimeNs] = strconv.FormatInt(event.RxTime.UnixNano(), 10)
	record[fieldEventType] = string(event.EventType)
	record[fieldData] = string(event.Data)
	return e.csvWriter.Write(record)
}
