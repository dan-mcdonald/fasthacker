package model

import (
	"encoding/json"
	"net/url"
	"time"
)

type UserID string
type ItemID int64

type User struct {
	ID        UserID    `json:"id"`
	Created   Time      `json:"created"`
	Karma     int       `json:"karma"`
	About     *string   `json:"about"`
	Submitted *[]ItemID `json:"submitted"`
}

type Item struct {
	ID          ItemID    `json:"id"`
	Deleted     *bool     `json:"deleted"`
	Type        string    `json:"type"`
	By          *UserID   `json:"by"`
	Time        Time      `json:"time"`
	Text        *string   `json:"text"`
	Dead        *bool     `json:"dead"`
	Parent      *ItemID   `json:"parent"`
	Kids        *[]ItemID `json:"kids"`
	URL         *string   `json:"url"`
	Score       *int      `json:"score"`
	Title       *string   `json:"title"`
	Parts       *[]ItemID `json:"parts"`
	Descendants *int      `json:"descendants"`
}

type TopStories []ItemID

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var i int64
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	t.Time = time.Unix(i, 0)
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Unix())
}

func (s *Item) Site() (string, error) {
	parsedUrl, err := url.Parse(*s.URL)
	if err != nil {
		return "", err
	}
	return parsedUrl.Hostname(), nil
}
