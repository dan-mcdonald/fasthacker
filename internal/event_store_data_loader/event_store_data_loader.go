package eventstoredataloader

import (
	"fmt"

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
	return nil, fmt.Errorf("not implemented")
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
