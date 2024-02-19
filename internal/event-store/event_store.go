package eventstore

import "github.com/dan-mcdonald/fasthacker/internal/model"

type EventStoreResponse struct {
	Item *model.Item
	Err  error
}

type eventStoreRequestType int

const (
	EventStoreRequestTypeGetLatestItem eventStoreRequestType = iota
)

type EventStoreRequest struct {
	ReqType eventStoreRequestType
	ID      model.ItemID
	Resp    chan EventStoreResponse
}

type EventStore struct {
	req chan EventStoreRequest
}

func NewEventStore(req chan EventStoreRequest) *EventStore {
	return &EventStore{req}
}

func (eq *EventStore) GetLatestItem(id model.ItemID) (*model.Item, error) {
	respCh := make(chan EventStoreResponse)
	eq.req <- EventStoreRequest{ReqType: EventStoreRequestTypeGetLatestItem, ID: id, Resp: respCh}
	resp := <-respCh
	return resp.Item, resp.Err
}
