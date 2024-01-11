package eventlog

// Interface for reading and writing events to a CSV log
// The CSV log has the following format:
// unix_nanoseconds, event_type, data
// The CSV log is append-only
// The CSV log is not thread-safe

import (
	"log"
	"time"

	"github.com/dan-mcdonald/fasthacker/internal/model"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

type EventType string

const (
	TypeItem = "item"
	TypeUser = "user"
)

type itemEvent struct {
	ID     uint64       `gorm:"primaryKey;autoIncrement:true"`
	RxTime time.Time    `gorm:"uniqueIndex:idx_rxtime_itemid"`
	ItemID model.ItemID `gorm:"uniqueIndex:idx_rxtime_itemid"`
	Data   []byte
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

	err = db.AutoMigrate(&itemEvent{})
	if err != nil {
		return nil, err
	}

	return &EventLog{
		db: db,
	}, nil
}

func (e *EventLog) ItemStream() chan model.ItemUpdate {
	ch := make(chan model.ItemUpdate, 100)
	go func() {
		defer close(ch)
		var batch []itemEvent
		result := e.db.Where("data IS NOT NULL").FindInBatches(&batch, 100, func(tx *gorm.DB, batchNum int) error {
			for _, record := range batch {
				ch <- model.ItemUpdate{
					RxTime: record.RxTime,
					ID:     record.ItemID,
					Data:   record.Data,
				}
			}
			return nil
		})
		if result.Error != nil {
			panic(result.Error)
		}
		log.Printf("eventlog: stream finished with %d events\n", result.RowsAffected)
	}()
	return ch
}

// Write an event to the log
func (e *EventLog) Write(updates []model.ItemUpdate) error {
	events := make([]itemEvent, len(updates))
	for i, update := range updates {
		events[i] = itemEvent{
			RxTime: update.RxTime,
			ItemID: update.ID,
			Data:   update.Data,
		}
	}
	return e.db.Create(events).Error
}
