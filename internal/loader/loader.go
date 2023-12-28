package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
	"github.com/dan-mcdonald/fasthacker/internal/model"
)

type DataLoader interface {
	GetTopStories() (model.TopStories, error)
	GetStory(id model.ItemID) (model.Item, error)
	GetComment(id model.ItemID) (model.Item, error)
}

type FirebaseNewsDataLoader struct {
	c http.Client
}

type CachingDataLoader struct {
	delegate     DataLoader
	storyCache   *cache.Cache[model.ItemID, model.Item]
	commentCache *cache.Cache[model.ItemID, model.Item]
	topStories   *model.TopStories
}

func NewLoader(ctx context.Context) DataLoader {
	return CachingDataLoader{
		delegate: FirebaseNewsDataLoader{
			c: http.Client{
				Timeout: 15 * time.Second,
			},
		},
		storyCache:   cache.NewContext[model.ItemID, model.Item](ctx),
		commentCache: cache.NewContext[model.ItemID, model.Item](ctx),
		topStories:   nil,
	}
}

func (c CachingDataLoader) GetTopStories() (model.TopStories, error) {
	if c.topStories != nil {
		log.Println("CachingDataLoader.GetTopStories: cache hit")
		return *c.topStories, nil
	}
	log.Println("CachingDataLoader.GetTopStories: cache miss")
	topStories, err := c.delegate.GetTopStories()
	return topStories, err
}

func (c CachingDataLoader) GetStory(id model.ItemID) (model.Item, error) {
	story, ok := c.storyCache.Get(id)
	if ok {
		log.Printf("CachingDataLoader.GetStory(%d): cache hit", id)
		return story, nil
	}
	log.Printf("CachingDataLoader.GetStory(%d): cache miss", id)
	story, err := c.delegate.GetStory(id)
	if err != nil {
		return model.Item{}, err
	}
	c.storyCache.Set(id, story)
	return story, nil
}

func (c CachingDataLoader) GetComment(id model.ItemID) (model.Item, error) {
	comment, ok := c.commentCache.Get(id)
	if ok {
		log.Printf("CachingDataLoader.GetComment(%d): cache hit", id)
		return comment, nil
	}
	log.Printf("CachingDataLoader.GetComment(%d): cache miss", id)
	comment, err := c.delegate.GetComment(id)
	if err != nil {
		return model.Item{}, err
	}
	c.commentCache.Set(id, comment)
	return comment, nil
}

func (fb FirebaseNewsDataLoader) GetTopStories() (model.TopStories, error) {
	resp, err := fb.c.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jsonDecoder := json.NewDecoder(resp.Body)
	var topStories model.TopStories
	err = jsonDecoder.Decode(&topStories)
	return topStories, err
}

func (fb FirebaseNewsDataLoader) GetStory(id model.ItemID) (model.Item, error) {
	resp, err := fb.c.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
	if err != nil {
		return model.Item{}, err
	}
	defer resp.Body.Close()
	jsonDecoder := json.NewDecoder(resp.Body)
	var story model.Item
	err = jsonDecoder.Decode(&story)
	if err != nil {
		return model.Item{}, err
	}
	return story, nil
}

func (fb FirebaseNewsDataLoader) GetComment(id model.ItemID) (model.Item, error) {
	resp, err := fb.c.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
	if err != nil {
		return model.Item{}, err
	}
	defer resp.Body.Close()
	jsonDecoder := json.NewDecoder(resp.Body)
	var comment model.Item
	err = jsonDecoder.Decode(&comment)
	if err != nil {
		return model.Item{}, err
	}
	return comment, nil
}
