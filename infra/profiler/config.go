package profiler

const (
	Enabled = iota
	Disabled
)

type Config struct {
	Addr                 string
	MutexProfileFraction int
	BlockProfileRate     int
}
