package pgw

import "testing"

func TestPoolState_ClosedIsAbsorbing(t *testing.T) {
	state := poolState{
		state:     HostStateError,
		closeChan: make(chan struct{}),
	}

	state.Close()
	if got := state.Get(); got != HostStateClosed {
		t.Fatalf("after Close: state = %s, want %s", got, HostStateClosed)
	}

	state.Set(HostStateConnected)
	state.Set(HostStateError)
	state.Set(HostStateRetrying)
	if got := state.Get(); got != HostStateClosed {
		t.Fatalf("after Set on closed: state = %s, want %s", got, HostStateClosed)
	}
}
