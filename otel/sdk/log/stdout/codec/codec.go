package codec

import (
	"io"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/sdk/log"
)

type Encoder interface {
	Encode(log.Record) error
}

type CodecFunc func(out io.Writer, opts ...Option) Encoder

// Options for encoding
type Options struct {
	// TimeStamp format layout
	// To disable timestamp(s) output: just leave it empty
	// https://pkg.go.dev/time#pkg-constants
	TimeStamp string
	// PrettyPrint indent string
	// To disable - leave it empty
	PrettyPrint string
}

type Option func(*Options)

func WithTimestamps(layout string) Option {
	return func(conf *Options) {
		// if layout == "" {
		// 	// disable. no output
		// }
		if layout != "" && !TimeStampIsValid(layout, time.Second) {
			// invalid layout spec ; -or- time(s) difference with layout encoding is greater-or-equal 1 second
			panic(errors.Errorf("otel/log/stdout/codec.Option( timestamp: %q ); invalid spec", layout))
		}
		conf.TimeStamp = layout
	}
}

func WithoutTimestamps() Option {
	return WithTimestamps("")
}

func WithPrittyPrint(indent string) Option {
	return func(conf *Options) {
		for _, c := range indent {
			if !unicode.IsSpace(c) {
				panic(errors.Errorf("otel/log/stdout/codec.Option( indent: %q ); accept whitespace(s)", indent))
			}
		}
		conf.PrettyPrint = indent
	}
}

func NewOptions(opts ...Option) Options {
	conf := Options{
		// defaults
		PrettyPrint: "",        // disabled
		TimeStamp:   TimeStamp, // enabled
	}
	for _, opt := range opts {
		opt(&conf)
	}
	return conf
}

var (
	regedit  sync.Mutex
	registry = make(map[string]CodecFunc)
)

func Register(codec string, build CodecFunc) {
	input := codec
	codec = strings.TrimSpace(codec)
	codec = strings.ToLower(codec)
	if codec != input {
		panic(errors.Errorf("otel/log/stdout/codec.Register( name: %q ); invalid", codec))
	}
	if codec == "" {
		panic(errors.Errorf("otel/log/stdout/codec.Register( name: ? ); required"))
	}
	if build == nil {
		panic(errors.Errorf("otel/log/stdout/codec.Register( name: %q ); not implemented", codec))
	}

	regedit.Lock()
	defer regedit.Unlock()
	if _, exists := registry[codec]; exists {
		panic(errors.Errorf("otel/log/stdout/codec.Register( name: %q ); duplicate", codec))
	}
	registry[codec] = build
}

func NewCodec(name string, out io.Writer, opts ...Option) Encoder {
	name = strings.ToLower(name)
	regedit.Lock()
	codec := registry[name]
	regedit.Unlock()
	if codec == nil {
		panic(errors.Errorf("otel/log/stdout/format.Codec( name: %q ); not registered", name))
	}
	return codec(out, opts...)
}
