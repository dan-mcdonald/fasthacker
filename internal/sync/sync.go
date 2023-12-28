package sync

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dan-mcdonald/fasthacker/internal/model"
	sse "github.com/r3labs/sse/v2"
)

type Sync struct {
	DBPath          string
	maxItemWritten  model.ItemID
	newMaxItemKnown chan model.ItemID
	csvWriter       *csv.Writer
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
		s.newMaxItemKnown <- maxitemPutData.MaxItem
	case "keep-alive":
		break
	default:
		fmt.Printf("sync: maxitem unknown event: %s\n", msgEvent)
	}
}

type UpdatePutData struct {
	ItemIDs []model.ItemID `json:"items"`
	UserIDs []model.UserID `json:"profiles"`
}

func (s *Sync) handleUpdateEvent(msg *sse.Event) {
	msgEvent := string(msg.Event[:])
	switch msgEvent {
	case "put":
		jsonDecoder := json.NewDecoder(bytes.NewReader(msg.Data))
		var updatePutData UpdatePutData
		if err := jsonDecoder.Decode(&updatePutData); err != nil {
			fmt.Printf("sync.handleUpdateEvent: error decoding update put data: %v", err)
		}
		fmt.Printf("sync: update got %d items and %d profiles\n", len(updatePutData.ItemIDs), len(updatePutData.UserIDs))
		for _, itemId := range updatePutData.ItemIDs {
			go requestItemChan(itemId, nil, nil)
		}
		for _, userId := range updatePutData.UserIDs {
			go requestUserChan(userId, nil, nil)
		}
	case "keep-alive":
		break
	default:
		fmt.Printf("sync: maxitem unknown event: %s\n", msgEvent)
	}
}

type MinimalItem struct {
	ID model.ItemID `json:"id"`
}

func (s *Sync) dbInit() error {
	fileHandle, err := os.OpenFile(s.DBPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	csvReader := csv.NewReader(fileHandle)
	csvReader.FieldsPerRecord = 1
	csvReader.ReuseRecord = true
	rowNumber := model.ItemID(0)
	for {
		rowNumber++
		record, err := csvReader.Read()
		if err == io.EOF {
			rowNumber--
			break
		}
		if err != nil {
			return err
		}
		jsonDecoder := json.NewDecoder(bytes.NewReader([]byte(record[0])))
		var minimalItem MinimalItem
		if err := jsonDecoder.Decode(&minimalItem); err != nil {
			return err
		}
		if minimalItem.ID != rowNumber {
			return fmt.Errorf("sync.dbInit: item at row %d ID was unexpected: %s", rowNumber, record[0])
		}
	}
	s.maxItemWritten = model.ItemID(rowNumber)
	s.csvWriter = csv.NewWriter(fileHandle)
	return nil
}

func requestItem(itemID model.ItemID) (ItemUpdate, error) {
	fmt.Printf("sync: requesting item %d\n", itemID)
	resp, err := http.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", itemID))
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
		Data:   buf.Bytes(),
	}, nil
}

type DataUpdate struct {
	RxTime time.Time
	Data   []byte
}

type UserUpdate DataUpdate
type ItemUpdate DataUpdate

func requestUser(userID model.UserID) (UserUpdate, error) {
	fmt.Printf("requesting user %s\n", userID)
	resp, err := http.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/user/%s.json", userID))
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
		Data:   buf.Bytes(),
	}, nil
}

func requestItemChan(itemId model.ItemID, ch chan ItemUpdate, errCh chan model.ItemID) {
	itemData, err := requestItem(itemId)
	if err != nil {
		fmt.Printf("sync.requestItemChan: error requesting item %d: %v\n", itemId, err)
		errCh <- itemId
		return
	}
	ch <- itemData
}

func requestUserChan(userId model.UserID, ch chan UserUpdate, errCh chan model.UserID) {
	itemData, err := requestUser(userId)
	if err != nil {
		fmt.Printf("error requesting item %s: %v\n", userId, err)
		errCh <- userId
		return
	}
	ch <- itemData
}

func (s *Sync) itemRequesterInit() error {
	go func() {
		itemUpdateCh := make(chan ItemUpdate, 1)
		errCh := make(chan model.ItemID, 1)
		maxItemKnown := s.maxItemWritten

		updateRequestsInFlight := func() {
			for i := s.maxItemWritten + 1; i <= maxItemKnown; i++ {
				if len(itemTable) >= 10 {
					break
				}
				if _, ok := itemTable[i]; ok {
					continue
				}
				itemTable[i] = make(chan ItemUpdate, 1)
				go requestItemChan(i, itemTable[i], errCh)
			}
		}

		for {
			select {
			case maxItemKnown = <-s.newMaxItemKnown:
				updateRequestsInFlight()
			case itemUpdate := <-itemTable[s.maxItemWritten+1]:
				delete(itemTable, s.maxItemWritten+1)
				err := s.csvWriter.Write([]string{
					strconv.FormatInt(itemUpdate.RxTime.Unix(), 10),
					string(itemUpdate.Data)},
				)
				if err != nil {
					log.Fatalf("sync.itemRequesterInit: error writing item %d: %v\n", s.maxItemWritten+1, err)
					continue
				}
				s.csvWriter.Flush()
				err = s.csvWriter.Error()
				if err != nil {
					log.Fatalf("sync.itemRequesterInit: error flushing csv writer: %v\n", err)
				}
				s.maxItemWritten++
				fmt.Printf("sync: wrote item %d\n", s.maxItemWritten)
				updateRequestsInFlight()
			case itemId := <-errCh:
				fmt.Printf("sync: retrying item: %d\n", itemId)
				go requestItemChan(itemId, itemTable[itemId], errCh)
			}
		}
	}()
	return nil
}

func (s *Sync) maxItemListenerInit() error {
	client := sse.NewClient("https://hacker-news.firebaseio.com/v0/maxitem.json")
	client.Headers
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

// Run runs the sync.
func (s *Sync) Run() error {
	s.newMaxItemKnown = make(chan model.ItemID, 1)

	err := s.dbInit()
	if err != nil {
		return err
	}
	fmt.Printf("sync: db initialized, max item written: %d\n", s.maxItemWritten)

	err = s.itemRequesterInit()
	if err != nil {
		return err
	}

	err = s.maxItemListenerInit()
	if err != nil {
		return err
	}

	err = s.updateListenerInit()
	if err != nil {
		return err
	}

	return nil
}
