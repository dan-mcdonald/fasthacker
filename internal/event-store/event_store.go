package eventstore

import "github.com/dan-mcdonald/fasthacker/internal/model"

type GetItemResponse struct {
	Item *model.Item
	Err  error
}

type GetItemRequest struct {
	ID   model.ItemID
	Resp chan GetItemResponse
}

type GetTopStoriesResponse struct {
	TopStories *model.TopStories
	Err        error
}

type GetTopStoriesRequest struct {
	Resp chan GetTopStoriesResponse
}

type EventStore struct {
	GetItemReq       chan GetItemRequest
	GetTopStoriesReq chan GetTopStoriesRequest
}

func NewEventStore() *EventStore {
	return &EventStore{
		GetItemReq:       make(chan GetItemRequest),
		GetTopStoriesReq: make(chan GetTopStoriesRequest),
	}
}

func (es *EventStore) GetLatestItem(id model.ItemID) (*model.Item, error) {
	respCh := make(chan GetItemResponse)
	es.GetItemReq <- GetItemRequest{ID: id, Resp: respCh}
	resp := <-respCh
	return resp.Item, resp.Err
}

func (es *EventStore) GetTopStories() (*model.TopStories, error) {
	respCh := make(chan GetTopStoriesResponse)
	es.GetTopStoriesReq <- GetTopStoriesRequest{Resp: respCh}
	resp := <-respCh
	return resp.TopStories, resp.Err
}
