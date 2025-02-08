package main

type Queue[T any] struct {
	list []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{list: make([]T, 0)}
}

// enqueue -> inserted idx
func (q *Queue[T]) Add(it T) int {
	q.list = append(q.list, it)
	return len(q.list) - 1
}

// dequeue -> it, exists
func (q *Queue[T]) Get() (T, bool) {
	var it T
	if len(q.list) > 0 {
		it = q.list[0]
		q.list = q.list[1:]
		return it, true
	}

	return it, false
}

func (q Queue[T]) Len() int {
	return len(q.list)
}

// if idx is invalid, it will return empty item
func (q Queue[T]) Peek(idx int) T {
	var it T
	if len(q.list) > idx {
		it = q.list[idx]
	}

	return it
}
