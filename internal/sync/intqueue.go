package sync

import (
	"golang.org/x/exp/constraints"
)

type intSpan[T constraints.Integer] struct {
	lower, upper T
}

type IntQueue[T constraints.Integer] struct {
	spans []intSpan[T]
}

func (q *IntQueue[T]) Enqueue(lower, upper T) {
	q.spans = append(q.spans, intSpan[T]{lower, upper})
}

func (q *IntQueue[T]) Pop() (T, bool) {
	if len(q.spans) == 0 {
		return 0, false
	}
	span := q.spans[0]
	v := span.lower
	if span.lower == span.upper {
		q.spans = q.spans[1:]
	} else {
		q.spans[0].lower++
	}
	return v, true
}

func (q *IntQueue[T]) Remove(v T) bool {
	for i, span := range q.spans {
		if span.lower <= v && v <= span.upper {
			if span.lower == span.upper {
				q.spans = append(q.spans[:i], q.spans[i+1:]...)
			} else {
				if span.lower == v {
					q.spans[i].lower++
				} else if span.upper == v {
					q.spans[i].upper--
				} else {
					q.spans = append(q.spans[:i], append([]intSpan[T]{{span.lower, v - 1}, {v + 1, span.upper}}, q.spans[i+1:]...)...)
				}
			}
			return true
		}
	}
	return false
}
