package main

import (
	"encoding/json"
	"html/template"
	"net/url"
	"time"
)

type CommentID int

type Comment struct {
	SubmitterName  string `json:"by"`
	ID             CommentID
	Kids           []CommentID
	Text           template.HTML
	SubmissionTime Time `json:"time"`
}

type StoryID int
type TopStories []StoryID

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

type Story struct {
	Title          string
	URL            string
	Score          int
	SubmitterName  string `json:"by"`
	SubmissionTime Time   `json:"time"`
	ID             int
	CommentCount   int `json:"descendants"`
	Kids           []CommentID
}

func (s *Story) Site() (string, error) {
	parsedUrl, err := url.Parse(s.URL)
	if err != nil {
		return "", err
	}
	return parsedUrl.Hostname(), nil
}
