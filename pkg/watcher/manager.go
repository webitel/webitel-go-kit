package watcher

type Manager interface {
	AddWatcher(clusterId string, watcher Watcher)
	RemoveCluster(clusterId string)
	GetCluster(clusterId string) []Watcher
	Notify(clusterId string, et EventType, data WatchMarshaller) error
	Enable()
	Disable()
	GetState() bool
}

type DefaultWatcherManager struct {
	clusters map[string][]Watcher
	state    bool
}

func NewDefaultWatcherManager(state bool) *DefaultWatcherManager {
	return &DefaultWatcherManager{
		clusters: make(map[string][]Watcher),
		state:    state,
	}
}

func (d *DefaultWatcherManager) AddWatcher(clusterId string, watcher Watcher) {
	d.clusters[clusterId] = append(d.clusters[clusterId], watcher)
}

func (d *DefaultWatcherManager) RemoveCluster(clusterId string) {
	delete(d.clusters, clusterId)
}

func (d *DefaultWatcherManager) GetCluster(clusterId string) []Watcher {
	return d.clusters[clusterId]
}

func (d *DefaultWatcherManager) Notify(clusterId string, et EventType, data WatchMarshaller) error {
	if !d.state {
		return nil
	}
	cluster := d.GetCluster(clusterId)
	if cluster == nil {
		return nil
	}
	for _, watcher := range cluster {
		if err := watcher.OnEvent(et, data); err != nil {
			return err
		}
	}
	return nil
}

func (d *DefaultWatcherManager) Enable() {
	d.state = true
}

func (d *DefaultWatcherManager) Disable() {
	d.state = false
}

func (d *DefaultWatcherManager) GetState() bool {
	return d.state
}
