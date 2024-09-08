package text

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"sync"

	logv "github.com/webitel/webitel-go-kit/otel/log"
	"github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec"
	"go.opentelemetry.io/otel/log"
	sdk "go.opentelemetry.io/otel/sdk/log"
)

type Encoder struct {
	opts codec.Options
	out  io.Writer
}

var _ codec.Encoder = (*Encoder)(nil)

func NewCodec(w io.Writer, opts ...codec.Option) codec.Encoder {
	return &Encoder{
		opts: codec.NewOptions(opts...),
		out:  w,
	}
}

func (enc *Encoder) Encode(rec sdk.Record) error {
	// panic("not implemented")
	state, free := alloc()
	defer free()

	if date := enc.opts.Timestamp(
		rec.Timestamp(),
	); date != nil {
		_, err := fmt.Fprintf(state, "%s ", date)
		if err != nil {
			return err
		}
	}

	level := rec.SeverityText()
	if level == "" {
		// level = rec.Severity().String()
		level = logv.Severity(rec.Severity()).String()
	}
	scope := rec.InstrumentationScope()
	name := path.Base(scope.Name)

	_, err := fmt.Fprintf(
		state, "%-7s %-6s ",
		("[" + level + "]"), (name + ":"),
	)
	if err != nil {
		return err
	}

	printValue(state, rec.Body())
	// body := rec.Body().String()
	// _, err := fmt.Fprintf(state, "[%s] %v", level, body)
	// if err != nil {
	// 	return err
	// }

	rec.WalkAttributes(func(att log.KeyValue) bool {
		err := printAttr(state, att)
		return err == nil
	})

	err = state.WriteByte('\n')
	if err != nil {
		return err
	}

	_, err = state.WriteTo(enc.out)
	return err
}

var statepool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 255))
	},
}

func alloc() (state *bytes.Buffer, free func()) {
	state = statepool.Get().(*bytes.Buffer)
	free = func() {
		state.Reset()
		statepool.Put(state)
	}
	return
}

func printAttr(w io.Writer, e log.KeyValue) (err error) {
	_, err = fmt.Fprintf(w, " ; %s=", e.Key)
	if err != nil {
		return err
	}
	return printValue(w, e.Value)
}

func printValue(w io.Writer, v log.Value) (err error) {
	switch v.Kind() {
	case log.KindBool:
		{
			_, err = fmt.Fprintf(w, "%t", v.AsBool())
		}
	case log.KindFloat64:
		{
			_, err = fmt.Fprintf(w, "%f", v.AsFloat64())
		}
	case log.KindInt64:
		{
			_, err = fmt.Fprintf(w, "%d", v.AsInt64())
		}
	case log.KindString:
		{
			_, err = fmt.Fprintf(w, "%s", v.AsString())
		}
	case log.KindBytes:
		{
			_, err = fmt.Fprintf(w, "%x", v.AsBytes())
		}
	case log.KindSlice:
		{
			sep := func(c byte) {
				_, err = w.Write([]byte{c})
			}
			if sep('['); err != nil {
				return // err
			}
			for i, v := range v.AsSlice() {
				if i > 0 {
					if sep(','); err != nil {
						return // err
					}
				}
				err = printValue(w, v)
				if err != nil {
					return // err
				}
			}
			if sep(']'); err != nil {
				return // err
			}
		}
	case log.KindMap:
		{
			print := func(bin ...byte) {
				_, err = w.Write(bin)
			}
			if print('{', ' '); err != nil {
				return // err
			}
			for i, v := range v.AsMap() {
				if i > 0 {
					if print(',', ' '); err != nil {
						return // err
					}
				}
				_, err = fmt.Fprintf(w, "%s=", v.Key)
				if err != nil {
					return // err
				}
				err = printValue(w, v.Value)
				if err != nil {
					return // err
				}
			}
			if print(' ', '}'); err != nil {
				return // err
			}
		}
	// case log.KindEmpty:
	default:
		_, err = fmt.Fprintf(w, "%s", v.AsString())
	}
	return // err?
}

func init() {
	codec.Register("text", NewCodec)
}
