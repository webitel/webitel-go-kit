package metric

import (
	"context"
	"sync"
	"testing"
)

func TestRegisterConcurrently(t *testing.T) {
	names := make([]string, 16)
	for i := range names {
		names[i] = scheme("race")
	}

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(2)
		go func() {
			defer wg.Done()
			Register(name, noopOptions)
		}()
		go func() {
			defer wg.Done()
			_, _ = NewOptions(context.Background(), name)
		}()
	}
	wg.Wait()

	for _, name := range names {
		if _, err := NewOptions(context.Background(), name); err != nil {
			t.Fatal(err)
		}
	}
}
