package profiler

const (
	Disabled = iota
	Enabled
)

type Config struct {
	Addr                 string
	MutexProfileFraction int
	BlockProfileRate     int
}
