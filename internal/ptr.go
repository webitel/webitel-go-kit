package internal

func ValueToPTR[T any](value T) *T {
	return &value
}
