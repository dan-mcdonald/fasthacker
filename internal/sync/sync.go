package sync

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	sse "github.com/r3labs/sse/v2"
)

type ItemID int64

type Sync struct {
	DBPath          string
	maxItemWritten  ItemID
	newMaxItemKnown chan ItemID
	csvWriter       *csv.Writer
}

type MaxitemPutData struct {
	MaxItem ItemID `json:"data"`
}

func (s *Sync) handleMessage(msg *sse.Event) {
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
	case "patch":
		fmt.Printf("sync: maxitem patch: %s\n", string(msg.Data[:]))
	case "keep-alive":
		break
	default:
		fmt.Printf("sync: maxitem unknown event: %s\n", msgEvent)
	}
}

type MinimalItem struct {
	ID ItemID `json:"id"`
}

func (s *Sync) dbInit() error {
	fileHandle, err := os.OpenFile(s.DBPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	csvReader := csv.NewReader(fileHandle)
	csvReader.FieldsPerRecord = 1
	csvReader.ReuseRecord = true
	rowNumber := ItemID(0)
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
	s.maxItemWritten = ItemID(rowNumber)
	s.csvWriter = csv.NewWriter(fileHandle)
	return nil
}

func requestItem(itemID ItemID) ([]byte, error) {
	fmt.Printf("sync: requesting item %d\n", itemID)
	resp, err := http.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", itemID))
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func (s *Sync) itemRequesterInit() error {
	go func() {
		lastMaxItemKnown := s.maxItemWritten
		for {
			newMaxItemKnown := <-s.newMaxItemKnown
			if newMaxItemKnown > lastMaxItemKnown {
				for itemID := lastMaxItemKnown + 1; itemID <= newMaxItemKnown; itemID++ {
					item, err := requestItem(itemID)
					if err != nil {
						fmt.Printf("sync.itemRequesterInit: error requesting item %d: %v\n", itemID, err)
						continue
					}
					err = s.csvWriter.Write([]string{string(item)})
					if err != nil {
						fmt.Printf("sync.itemRequesterInit: error writing item %d: %v\n", itemID, err)
						continue
					}
					s.csvWriter.Flush()
					err = s.csvWriter.Error()
					if err != nil {
						fmt.Printf("sync.itemRequesterInit: error flushing csv writer: %v\n", err)
					}
					fmt.Printf("sync: wrote item %d\n", itemID)
					s.maxItemWritten = itemID
				}
				lastMaxItemKnown = newMaxItemKnown
			}
		}
	}()
	return nil
}

func (s *Sync) maxItemListenerInit() error {
	client := sse.NewClient("https://hacker-news.firebaseio.com/v0/maxitem.json")
	client.OnConnect(func(c *sse.Client) {
		fmt.Println("sync: SSE maxitem connected")
	})
	client.OnDisconnect(func(c *sse.Client) {
		fmt.Println("sync: SSE maxitem disconnected")
	})
	client.Subscribe("", s.handleMessage)
	return nil
}

// Run runs the sync.
func (s *Sync) Run() error {
	s.newMaxItemKnown = make(chan ItemID, 1)

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

	return nil
}
