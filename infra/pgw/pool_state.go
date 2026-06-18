package pgw

import (
	"context"
	"sync"
)

type PoolState string

const (
	HostStateConnecting PoolState = "connecting"
	HostStateConnected  PoolState = "connected"
	HostStateRetrying   PoolState = "retrying"
	HostStateClosed     PoolState = "closed"
	HostStateError      PoolState = "error"
)

type HostMode string

const (
	HostModeAny       HostMode = "any"
	HostModeReadWrite HostMode = "read-write"
	HostModeReadOnly  HostMode = "read-only"
	HostModePrimary   HostMode = "primary"
	HostModeStandby   HostMode = "standby"
)

type poolState struct {
	state      PoolState
	mode       HostMode
	stateMu    sync.RWMutex
	channelsMu sync.RWMutex

	notifyChangeChannels []chan PoolState
	closeChan            chan struct{}
}

func (s *poolState) Set(state PoolState) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if state == s.state {
		return
	}

	s.state = state
	s.notifyChange()
}

func (s *poolState) Get() PoolState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

func (s *poolState) GetMode() HostMode {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.mode
}

func (s *poolState) SubscribeChange(ctx context.Context) <-chan PoolState {
	ch := make(chan PoolState, 4)

	s.channelsMu.Lock()
	s.notifyChangeChannels = append(s.notifyChangeChannels, ch)
	s.channelsMu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			s.channelsMu.Lock()
			defer s.channelsMu.Unlock()
			for i, c := range s.notifyChangeChannels {
				if c == ch {
					s.notifyChangeChannels = append(s.notifyChangeChannels[:i], s.notifyChangeChannels[i+1:]...)
					close(ch)
					return
				}
			}
		case <-s.closeChan:
			return
		}
	}()

	return ch
}

func (s *poolState) notifyChange() {
	s.channelsMu.RLock()
	defer s.channelsMu.RUnlock()
	for _, ch := range s.notifyChangeChannels {
		select {
		case ch <- s.state:
		default:
		}
	}
}

func (s *poolState) Close() {
	s.Set(HostStateClosed)

	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()
	close(s.closeChan)

	for _, ch := range s.notifyChangeChannels {
		close(ch)
	}

	s.notifyChangeChannels = nil
}
