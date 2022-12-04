package set

import "fmt"

func FromSlice[T comparable](items ...T) Set[T] {
	set := make(Set[T], len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

func New[T comparable](n int) Set[T] {
	return make(Set[T], n)
}

type Set[T comparable] map[T]struct{}

func (set Set[T]) Delete(t T) {
	delete(set, t)
}

func (set Set[T]) Contains(t T) bool {
	_, ok := set[t]
	return ok
}

func (set Set[T]) Clone() Set[T] {
	clone := New[T](set.Len())
	for item := range set {
		clone[item] = struct{}{}
	}
	return clone
}

func (set Set[T]) Len() int {
	return len(set)
}

func (set Set[T]) AsSlice() []T {
	slice := make([]T, 0, set.Len())
	for item := range set {
		slice = append(slice, item)
	}
	return slice
}

func (set Set[T]) String() string {
	return fmt.Sprintf("%s", map[T]struct{}(set))
}
