package eventstoredataloader

import (
	eventstore "github.com/dan-mcdonald/fasthacker/internal/event-store"
	"github.com/dan-mcdonald/fasthacker/internal/model"
)

type EventStoreDataLoader struct {
	es *eventstore.EventStore
}

func NewEventStoreDataLoader(es *eventstore.EventStore) *EventStoreDataLoader {
	return &EventStoreDataLoader{es: es}
}

func (esdl *EventStoreDataLoader) GetTopStories() (model.TopStories, error) {
	item, err := esdl.es.GetTopStories()
	if err != nil {
		return model.TopStories{}, err
	}
	return *item, nil
}

func (esdl *EventStoreDataLoader) GetStory(id model.ItemID) (model.Item, error) {
	item, err := esdl.es.GetLatestItem(id)
	if err != nil {
		return model.Item{}, err
	}
	return *item, nil
}

func (esdl *EventStoreDataLoader) GetComment(id model.ItemID) (model.Item, error) {
	item, err := esdl.es.GetLatestItem(id)
	if err != nil {
		return model.Item{}, err
	}
	return *item, nil
}
