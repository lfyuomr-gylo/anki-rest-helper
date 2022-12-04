package lang

func New[T any](t T) *T {
	return &t
}
