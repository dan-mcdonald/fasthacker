package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	eventlog "github.com/dan-mcdonald/fasthacker/internal/event-log"
	"github.com/dan-mcdonald/fasthacker/internal/model"
	"github.com/hashicorp/go-metrics"
	sse "github.com/r3labs/sse/v2"
)

type Sync struct {
	dbPath               string
	sink                 metrics.MetricSink
	itemSeen             chan model.ItemID
	neededItemsWorkQueue chan model.ItemID
	notifyItem           chan ItemUpdate
}

func NewSync(dbPath string, sink metrics.MetricSink) *Sync {
	return &Sync{
		dbPath: dbPath,
		sink:   sink,
	}
}

type FasthackerTransport struct{}

var customHeaders = map[string]string{
	"User-Agent": "fasthacker",
	"From":       "daniel@mcd.onald.net",
}

func (t *FasthackerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}

var httpClient = http.Client{
	Transport: &FasthackerTransport{},
	Timeout:   30 * time.Second,
}

type MaxitemPutData struct {
	MaxItem model.ItemID `json:"data"`
}

func (s *Sync) handleMaxItemEvent(msg *sse.Event) {
	msgEvent := string(msg.Event[:])
	switch msgEvent {
	case "put":
		jsonDecoder := json.NewDecoder(bytes.NewReader(msg.Data))
		var maxitemPutData MaxitemPutData
		if err := jsonDecoder.Decode(&maxitemPutData); err != nil {
			fmt.Printf("sync.handleMessage: error decoding maxitem put data: %v", err)
		}
		fmt.Printf("sync: new maxitem value %d\n", maxitemPutData.MaxItem)
		s.itemSeen <- maxitemPutData.MaxItem
	case "keep-alive":
		break
	default:
		fmt.Printf("sync: maxitem unknown event: %s\n", msgEvent)
	}
}

type UpdatePutMessage struct {
	Path string `json:"path"`
	Data struct {
		ItemIDs []model.ItemID `json:"items"`
		UserIDs []model.UserID `json:"profiles"`
	} `json:"data"`
}

func (s *Sync) handleUpdateEvent(msg *sse.Event) {
	msgEvent := string(msg.Event[:])
	switch msgEvent {
	case "put":
		jsonDecoder := json.NewDecoder(bytes.NewReader(msg.Data))
		var updatePutMsg UpdatePutMessage
		if err := jsonDecoder.Decode(&updatePutMsg); err != nil {
			fmt.Printf("sync.handleUpdateEvent: error decoding update put data: %v", err)
		}
		fmt.Printf("sync: update got %d items and %d profiles\n", len(updatePutMsg.Data.ItemIDs), len(updatePutMsg.Data.UserIDs))
		for _, itemID := range updatePutMsg.Data.ItemIDs {
			s.itemSeen <- itemID
		}
		// TODO handle profiles
	case "keep-alive":
		break
	default:
		fmt.Printf("sync: maxitem unknown event: %s\n", msgEvent)
	}
}

type MinimalItem struct {
	ID model.ItemID `json:"id"`
}

func requestItem(itemID model.ItemID) (ItemUpdate, error) {
	resp, err := httpClient.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", itemID))
	if err != nil {
		return ItemUpdate{}, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	rxTime := time.Now()
	if err != nil {
		return ItemUpdate{}, err
	}
	return ItemUpdate{
		RxTime: rxTime,
		ID:     itemID,
		Data:   buf.Bytes(),
	}, nil
}

type DataUpdate[T comparable] struct {
	RxTime time.Time
	ID     T
	Data   []byte
}

type UserUpdate DataUpdate[model.UserID]
type ItemUpdate DataUpdate[model.ItemID]

func requestUser(userID model.UserID) (UserUpdate, error) {
	fmt.Printf("requesting user %s\n", userID)
	resp, err := httpClient.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/user/%s.json", userID))
	if err != nil {
		return UserUpdate{}, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	rxTime := time.Now()
	if err != nil {
		return UserUpdate{}, err
	}
	return UserUpdate{
		RxTime: rxTime,
		ID:     userID,
		Data:   buf.Bytes(),
	}, nil
}

func (s *Sync) maxItemNotifierStart() error {
	client := sse.NewClient("https://hacker-news.firebaseio.com/v0/maxitem.json")
	client.OnConnect(func(c *sse.Client) {
		fmt.Println("sync: SSE maxitem connected")
	})
	client.OnDisconnect(func(c *sse.Client) {
		fmt.Println("sync: SSE maxitem disconnected")
	})
	client.Subscribe("", s.handleMaxItemEvent)
	return nil
}

func (s *Sync) updateListenerInit() error {
	client := sse.NewClient("https://hacker-news.firebaseio.com/v0/updates.json")
	client.OnConnect(func(c *sse.Client) {
		fmt.Println("sync: SSE updates connected")
	})
	client.OnDisconnect(func(c *sse.Client) {
		fmt.Println("sync: SSE updates disconnected")
	})
	client.Subscribe("", s.handleUpdateEvent)
	return nil
}

func (s *Sync) neededItemsQueueManager() {
	actualMaxItemID := model.ItemID(1)
	neededItems := make(map[model.ItemID]struct{})
	for {
		if len(neededItems) == 0 {
			lastItemSeen := <-s.itemSeen
			if lastItemSeen > actualMaxItemID {
				oldMaxItemID := actualMaxItemID
				actualMaxItemID = lastItemSeen
				for i := oldMaxItemID + 1; i <= actualMaxItemID; i++ {
					neededItems[i] = struct{}{}
				}
			}
		} else {
			var nextItem model.ItemID
			for itemID := range neededItems {
				nextItem = itemID
				break
			}
			select {
			case candidateMaxItem := <-s.itemSeen:
				if candidateMaxItem > actualMaxItemID {
					oldMaxItemID := actualMaxItemID
					actualMaxItemID = candidateMaxItem
					for i := oldMaxItemID + 1; i <= actualMaxItemID; i++ {
						neededItems[i] = struct{}{}
					}
				}
			case s.neededItemsWorkQueue <- nextItem:
				delete(neededItems, nextItem)
			}
		}
	}
}

func (s *Sync) getterWorker() {
	for itemID := range s.neededItemsWorkQueue {
		reqStart := time.Now()
		itemUpdate, err := requestItem(itemID)
		if err != nil {
			log.Fatalf("sync.getterWorker: error requesting item %d: %v\n", itemID, err)
		}
		s.sink.AddSample([]string{"request_item"}, float32(time.Since(reqStart).Milliseconds()))
		s.notifyItem <- itemUpdate
	}
}

func (s *Sync) startEventLogManager() error {
	eventLog, err := eventlog.NewEventLog(s.dbPath)
	if err != nil {
		return err
	}
	for event := range eventLog.Stream() {
		switch event.EventType {
		case eventlog.TypeItem:
			var item model.Item
			err = json.Unmarshal(event.Data, &item)
			if err != nil {
				return err
			}
			s.itemSeen <- item.ID
		case eventlog.TypeUser:
			panic("todo handle user object")
		default:
			log.Fatalf("unknown event type: %s", event.EventType)
		}
	}
	fmt.Println("sync: db initialized")

	go func() {
		defer eventLog.Close()
		for itemUpdate := range s.notifyItem {
			s.itemSeen <- itemUpdate.ID
			err := eventLog.Write(eventlog.Event{
				RxTime:    itemUpdate.RxTime,
				EventType: eventlog.TypeItem,
				Data:      itemUpdate.Data,
			})
			if err != nil {
				log.Fatalf("sync.Run: error writing item %d: %v\n", itemUpdate.Data, err)
			}
		}
	}()
	return nil
}

const worker_count = 100

// Start runs the sync.
func (s *Sync) Start() error {
	s.itemSeen = make(chan model.ItemID, worker_count)
	s.neededItemsWorkQueue = make(chan model.ItemID, worker_count)
	go s.neededItemsQueueManager()
	s.notifyItem = make(chan ItemUpdate, worker_count)

	var err error

	err = s.startEventLogManager()
	if err != nil {
		log.Fatalf("sync.Run: error starting event log manager: %v\n", err)
	}

	for i := 0; i < worker_count; i++ {
		go s.getterWorker()
	}

	err = s.updateListenerInit()
	if err != nil {
		return err
	}

	err = s.maxItemNotifierStart()
	if err != nil {
		return err
	}

	return nil
}
