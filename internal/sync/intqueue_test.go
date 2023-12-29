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
		x, ok := q.Pop()
		if !ok {
			t.Errorf("queue unexpectedly empty")
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, ok := q.Pop()
	if ok {
		t.Errorf("expected queue empty, got not empty")
	}
}

func TestIntQueueRemoveHead(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 10)
	q.Remove(1)
	expected := []int{2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, v := range expected {
		x, ok := q.Pop()
		if !ok {
			t.Errorf("queue unexpectedly empty")
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, ok := q.Pop()
	if ok {
		t.Errorf("expected queue empty, got not empty")
	}
}

func TestIntQueueRemoveTail(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 10)
	q.Remove(10)
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	for _, v := range expected {
		x, ok := q.Pop()
		if !ok {
			t.Errorf("queue unexpectedly empty")
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, ok := q.Pop()
	if ok {
		t.Errorf("expected queue empty, got not empty")
	}
}

func TestIntQueueRemove1Span(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 1)
	q.Remove(1)
	_, ok := q.Pop()
	if ok {
		t.Errorf("expected queue empty, got not empty")
	}
}

func TestIntQueueRemoveInvalid(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(1, 9)
	ok := q.Remove(10)
	if ok {
		t.Errorf("expected remove to fail")
	}
}

func TestEnqueueOutOfOrder(t *testing.T) {
	q := IntQueue[int]{}
	q.Enqueue(6, 10)
	q.Enqueue(1, 4)
	q.Enqueue(11, 12)
	expected := []int{1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12}
	for _, v := range expected {
		x, ok := q.Pop()
		if !ok {
			t.Errorf("queue unexpectedly empty")
		}
		if x != v {
			t.Errorf("expected %d, got %d", v, x)
		}
	}
	_, ok := q.Pop()
	if ok {
		t.Errorf("expected queue empty, got not empty")
	}
}
