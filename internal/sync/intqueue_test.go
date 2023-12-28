package sync

import (
	"testing"
)

func TestIntQueueRemoveMiddle(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 10)
	q.Remove(5)
	expected := []int{1, 2, 3, 4, 6, 7, 8, 9, 10}
	for _, v := range expected {
		x, err := q.Dequeue()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, err := q.Dequeue()
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestIntQueueRemoveHead(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 10)
	q.Remove(1)
	expected := []int{2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, v := range expected {
		x, err := q.Dequeue()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, err := q.Dequeue()
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestIntQueueRemoveTail(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 10)
	q.Remove(10)
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	for _, v := range expected {
		x, err := q.Dequeue()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, err := q.Dequeue()
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestIntQueueRemove1Span(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 1)
	q.Remove(1)
	_, err := q.Dequeue()
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestIntQueueRemoveInvalid(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 9)
	err := q.Remove(10)
	if err == nil {
		t.Errorf("expected error, got none")
	}
}
