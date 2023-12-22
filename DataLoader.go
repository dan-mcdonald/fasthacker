package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
)

type DataLoader interface {
	GetTopStories() (TopStories, error)
	GetStory(id StoryID) (Story, error)
	GetComment(id CommentID) (Comment, error)
}

type FirebaseNewsDataLoader struct {
	c http.Client
}

type CachingDataLoader struct {
	delegate     DataLoader
	storyCache   *cache.Cache[StoryID, Story]
	commentCache *cache.Cache[CommentID, Comment]
	topStories   *TopStories
}

func NewLoader(ctx context.Context) DataLoader {
	return CachingDataLoader{
		delegate: FirebaseNewsDataLoader{
			c: http.Client{
				Timeout: 15 * time.Second,
			},
		},
		storyCache:   cache.NewContext[StoryID, Story](ctx),
		commentCache: cache.NewContext[CommentID, Comment](ctx),
		topStories:   nil,
	}
}

func (c CachingDataLoader) GetTopStories() (TopStories, error) {
	if c.topStories != nil {
		log.Println("CachingDataLoader.GetTopStories: cache hit")
		return *c.topStories, nil
	}
	log.Println("CachingDataLoader.GetTopStories: cache miss")
	topStories, err := c.delegate.GetTopStories()
	return topStories, err
}

func (c CachingDataLoader) GetStory(id StoryID) (Story, error) {
	story, ok := c.storyCache.Get(id)
	if ok {
		log.Printf("CachingDataLoader.GetStory(%d): cache hit", id)
		return story, nil
	}
	log.Printf("CachingDataLoader.GetStory(%d): cache miss", id)
	story, err := c.delegate.GetStory(id)
	if err != nil {
		return Story{}, err
	}
	c.storyCache.Set(id, story)
	return story, nil
}

func (c CachingDataLoader) GetComment(id CommentID) (Comment, error) {
	comment, ok := c.commentCache.Get(id)
	if ok {
		log.Printf("CachingDataLoader.GetComment(%d): cache hit", id)
		return comment, nil
	}
	log.Printf("CachingDataLoader.GetComment(%d): cache miss", id)
	comment, err := c.delegate.GetComment(id)
	if err != nil {
		return Comment{}, err
	}
	c.commentCache.Set(id, comment)
	return comment, nil
}

func (fb FirebaseNewsDataLoader) GetTopStories() (TopStories, error) {
	resp, err := fb.c.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jsonDecoder := json.NewDecoder(resp.Body)
	var topStories TopStories
	err = jsonDecoder.Decode(&topStories)
	return topStories, err
}

func (fb FirebaseNewsDataLoader) GetStory(id StoryID) (Story, error) {
	resp, err := fb.c.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
	if err != nil {
		return Story{}, err
	}
	defer resp.Body.Close()
	jsonDecoder := json.NewDecoder(resp.Body)
	var story Story
	err = jsonDecoder.Decode(&story)
	if err != nil {
		return Story{}, err
	}
	return story, nil
}

func (fb FirebaseNewsDataLoader) GetComment(id CommentID) (Comment, error) {
	resp, err := fb.c.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
	if err != nil {
		return Comment{}, err
	}
	defer resp.Body.Close()
	jsonDecoder := json.NewDecoder(resp.Body)
	var comment Comment
	err = jsonDecoder.Decode(&comment)
	if err != nil {
		return Comment{}, err
	}
	return comment, nil
}
