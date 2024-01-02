package eventlog

// Interface for reading and writing events to a CSV log
// The CSV log has the following format:
// unix_nanoseconds, event_type, data
// The CSV log is append-only
// The CSV log is not thread-safe

import (
	"time"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
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
	db *gorm.DB
}

func (e *EventLog) Close() error {
	db, err := e.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

// NewEventLog creates a new event log
func NewEventLog(path string) (*EventLog, error) {
	db, err := gorm.Open(gormlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Event{})

	return &EventLog{
		db: db,
	}, nil
}

func (e *EventLog) Stream() chan Event {
	ch := make(chan Event, 100)
	go func() {
		defer close(ch)
		var batch []Event
		e.db.FindInBatches(&batch, 100, func(tx *gorm.DB, _ int) error {
			for _, record := range batch {
				ch <- Event{
					RxTime:    record.RxTime,
					EventType: record.EventType,
					Data:      record.Data,
				}
			}
			return nil
		})
	}()
	return ch
}

// Write an event to the log
func (e *EventLog) Write(event Event) error {
	return e.db.Create(event).Error
}
