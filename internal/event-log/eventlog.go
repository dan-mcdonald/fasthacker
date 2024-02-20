package eventlog

// Interface for reading and writing events to a CSV log
// The CSV log has the following format:
// unix_nanoseconds, event_type, data
// The CSV log is append-only
// The CSV log is not thread-safe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dan-mcdonald/fasthacker/internal/model"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

type topStoriesEvent struct {
	ID     uint64    `gorm:"primaryKey;autoIncrement:true"`
	RxTime time.Time `gorm:"uniqueIndex:idx_rxtime"`
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
	logger := logger.New(
		log.New(os.Stdout, "\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: 500 * time.Millisecond,
			LogLevel:      logger.Silent,
			Colorful:      true,
		},
	)
	db, err := gorm.Open(gormlite.Open(path), &gorm.Config{
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("eventlog: migration start")
	migrator := db.Debug().Migrator()
	if !migrator.HasTable(&itemEvent{}) {
		if err := migrator.CreateTable(&itemEvent{}); err != nil {
			return nil, err
		}
	}
	if !migrator.HasTable(&topStoriesEvent{}) {
		if err := migrator.CreateTable(&topStoriesEvent{}); err != nil {
			return nil, err
		}
	}
	fmt.Println("eventlog: migration complete")

	return &EventLog{
		db: db,
	}, nil
}

func (e *EventLog) ItemIDs() []model.ItemID {
	startTime := time.Now()
	defer func() {
		fmt.Printf("eventlog.ItemIDs took %v\n", time.Since(startTime))
	}()
	var itemIDs []model.ItemID
	tx := e.db.Model(&itemEvent{}).Pluck("item_id", &itemIDs)
	if tx.Error != nil {
		log.Fatalf("eventlog.ItemIDs: error reading item IDs: %v\n", tx.Error)
	}
	return itemIDs
}

// WriteItemBatch an item event to the log
func (e *EventLog) WriteItemBatch(updates []model.ItemUpdate) error {
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

func (e *EventLog) GetLatestItem(id model.ItemID) (*model.Item, error) {
	var event itemEvent
	tx := e.db.Where("item_id = ?", id).Order("rx_time DESC").First(&event)
	if tx.Error != nil {
		return nil, tx.Error
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(event.Data))
	var item model.Item
	if err := jsonDecoder.Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (e *EventLog) WriteTopStories(topStoriesUpdate model.TopStoriesUpdate) error {
	event := topStoriesEvent{
		RxTime: topStoriesUpdate.RxTime,
		Data:   topStoriesUpdate.Data,
	}
	return e.db.Create(&event).Error
}

func (e *EventLog) GetTopStories() (model.TopStories, error) {
	var event topStoriesEvent
	tx := e.db.Order("rx_time DESC").First(&event)
	if tx.Error != nil {
		return nil, tx.Error
	}
	var topStories model.TopStories
	jsonDecoder := json.NewDecoder(bytes.NewReader(event.Data))
	if err := jsonDecoder.Decode(&topStories); err != nil {
		return nil, err
	}
	return topStories, nil
}
