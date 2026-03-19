package profiler

type Flag int

const (
	Enabled Flag = iota
	Disabled
)

type Config struct {
	Addr                 string
	MutexProfileFraction Flag
	BlockProfileRate     Flag
}
