package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
	eventstore "github.com/dan-mcdonald/fasthacker/internal/event-store"
	eventstoredataloader "github.com/dan-mcdonald/fasthacker/internal/event_store_data_loader"
	"github.com/dan-mcdonald/fasthacker/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metrics = struct {
	GetItemCacheHitLatency        prometheus.Histogram
	GetItemCacheMissLatency       prometheus.Histogram
	GetTopStoriesCacheHitLatency  prometheus.Histogram
	GetTopStoriesCacheMissLatency prometheus.Histogram
}{
	GetItemCacheHitLatency: promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "fasthacker_get_item_cache_hit_latency_seconds",
		Help: "The latency of cache hits for GetItem",
	}),
	GetItemCacheMissLatency: promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "fasthacker_get_item_cache_miss_latency_seconds",
		Help: "The latency of cache misses for GetItem",
	}),
	GetTopStoriesCacheHitLatency: promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "fasthacker_get_top_stories_cache_hit_latency_seconds",
		Help: "The latency of cache hits for GetTopStories",
	}),
	GetTopStoriesCacheMissLatency: promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "fasthacker_get_top_stories_cache_miss_latency_seconds",
		Help: "The latency of cache misses for GetTopStories",
	}),
}

type DataLoader interface {
	GetTopStories() (model.TopStories, error)
	GetItem(id model.ItemID) (model.Item, error)
}

type FirebaseNewsDataLoader struct {
	c http.Client
}

type CachingDataLoader struct {
	delegate        DataLoader
	itemCache       *cache.Cache[model.ItemID, model.Item]
	topStoriesCache *cache.Cache[struct{}, model.TopStories]
}

func NewLoader(ctx context.Context, es *eventstore.EventStore) DataLoader {
	return CachingDataLoader{
		delegate:        eventstoredataloader.NewEventStoreDataLoader(es),
		itemCache:       cache.NewContext[model.ItemID, model.Item](ctx),
		topStoriesCache: cache.NewContext[struct{}, model.TopStories](ctx),
	}
}

func (c CachingDataLoader) GetTopStories() (model.TopStories, error) {
	start := time.Now()
	topStories, ok := c.topStoriesCache.Get(struct{}{})
	if ok {
		metrics.GetTopStoriesCacheHitLatency.Observe(time.Since(start).Seconds())
		return topStories, nil
	}
	topStories, err := c.delegate.GetTopStories()
	if err == nil {
		c.topStoriesCache.Set(struct{}{}, topStories, cache.WithExpiration(1*time.Minute))
		metrics.GetTopStoriesCacheMissLatency.Observe(time.Since(start).Seconds())
	}
	return topStories, err
}

func (c CachingDataLoader) GetItem(id model.ItemID) (model.Item, error) {
	start := time.Now()
	item, ok := c.itemCache.Get(id)
	if ok {
		metrics.GetItemCacheHitLatency.Observe(time.Since(start).Seconds())
		return item, nil
	}
	item, err := c.delegate.GetItem(id)
	if err == nil {
		c.itemCache.Set(id, item)
		metrics.GetItemCacheMissLatency.Observe(time.Since(start).Seconds())
	}
	return item, nil
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
