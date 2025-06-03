package text

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mattn/go-isatty"
	// logv "github.com/webitel/webitel-go-kit/infra/otel/log"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout/codec"
	"go.opentelemetry.io/otel/log"
	sdk "go.opentelemetry.io/otel/sdk/log"
)

type Encoder struct {
	opts codec.Options
	out  io.Writer
}

var _ codec.Encoder = (*Encoder)(nil)

func NewCodec(w io.Writer, opts ...codec.Option) codec.Encoder {
	enc := &Encoder{
		opts: codec.NewOptions(opts...),
		out:  w,
	}
	if !enc.opts.NoColor {
		file, is := w.(*os.File)
		enc.opts.NoColor =
			!(is && file != nil) ||
				!isatty.IsTerminal(
					file.Fd(),
				)
	}
	return enc
}

func (enc *Encoder) Encode(rec sdk.Record) error {
	// panic("not implemented")
	buf := newBuffer()
	defer buf.Free()
	noColor := enc.opts.NoColor

	if date := enc.opts.Timestamp(
		rec.Timestamp(),
	); date != nil {
		buf.WriteStringIf(!noColor, ansiFaint)
		*buf = date.Time.AppendFormat(*buf, date.Format)
		buf.WriteStringIf(!noColor, ansiReset)
		buf.WriteByte(' ')
	}

	levelVerb := rec.Severity()
	// levelText := rec.SeverityText()
	// if levelText == "" {
	// 	// level = rec.Severity().String()
	// 	levelText = logv.Severity(rec.Severity()).String()
	// }

	// scope := rec.InstrumentationScope()
	// name := path.Base(scope.Name)

	// buf.WriteString(name)
	// buf.WriteByte(':')
	// buf.WriteByte(' ')

	// func (h *handler) appendLevel(buf *buffer, level slog.Level) {
	appendLevelDelta := func(delta log.Severity) {
		if delta == 0 {
			return
		} else if delta > 0 {
			buf.WriteByte('+')
		}
		*buf = strconv.AppendInt(*buf, int64(delta), 10)
	}
	switch {
	// case levelVerb < log.SeverityDebug:
	// 	buf.WriteString("TRC")
	// 	appendLevelDelta(levelVerb - log.SeverityTrace)
	case levelVerb < log.SeverityInfo:
		// buf.WriteStringIf(!noColor, ansiFaint)
		buf.WriteStringIf(!noColor, ansiBrightYellow)
		buf.WriteString("DEBUG")
		appendLevelDelta(levelVerb - log.SeverityDebug)
		buf.WriteStringIf(!noColor, ansiReset)
	case levelVerb < log.SeverityWarn:
		buf.WriteStringIf(!noColor, ansiBrightGreen)
		buf.WriteString("INFO")
		appendLevelDelta(levelVerb - log.SeverityInfo)
		buf.WriteStringIf(!noColor, ansiReset)
	case levelVerb < log.SeverityError:
		// buf.WriteStringIf(!noColor, ansiBrightYellow)
		buf.WriteStringIf(!noColor, ansiBrightRed)
		buf.WriteString("WARN")
		appendLevelDelta(levelVerb - log.SeverityWarn)
		buf.WriteStringIf(!noColor, ansiReset)
	case levelVerb < log.SeverityFatal:
		buf.WriteStringIf(!noColor, ansiBrightRed)
		buf.WriteString("ERROR")
		appendLevelDelta(levelVerb - log.SeverityError)
		buf.WriteStringIf(!noColor, ansiReset)
	default:
		buf.WriteStringIf(!noColor, ansiBrightRed)
		buf.WriteString("FATAL")
		appendLevelDelta(levelVerb - log.SeverityFatal)
		buf.WriteStringIf(!noColor, ansiReset)
	}
	// }
	buf.WriteByte(' ')
	enc.printValue(buf, rec.Body())
	// body := rec.Body().String()
	// _, err := fmt.Fprintf(state, "[%s] %v", level, body)
	// if err != nil {
	// 	return err
	// }

	rec.WalkAttributes(func(att log.KeyValue) bool {
		err := enc.printAttr(buf, att)
		return err == nil
	})

	err := buf.WriteByte('\n')
	if err != nil {
		return err
	}

	// _, err = state.WriteTo(enc.out)
	_, err = enc.out.Write(*buf)
	return err
}

func (enc *Encoder) printAttr(buf *buffer, att log.KeyValue) (err error) {
	// complex
	if att.Value.Kind() == log.KindMap {
		prefix := att.Key
		for _, att := range att.Value.AsMap() {
			att.Key = prefix + "." + att.Key
			err = enc.printAttr(buf, att)
			if err != nil {
				return err
			}
		}
		return
	}
	// scalar
	buf.WriteStringIf(!enc.opts.NoColor, ansiFaint)
	_, err = fmt.Fprintf(buf, " ; %s=", att.Key)
	buf.WriteStringIf(!enc.opts.NoColor, ansiReset)
	if err != nil {
		return err
	}
	return enc.printValue(buf, att.Value)
}

func (enc *Encoder) printValue(buf *buffer, val log.Value) (err error) {
	switch val.Kind() {
	case log.KindBool:
		{
			_, err = fmt.Fprintf(buf, "%t", val.AsBool())
		}
	case log.KindFloat64:
		{
			_, err = fmt.Fprintf(buf, "%f", val.AsFloat64())
		}
	case log.KindInt64:
		{
			_, err = fmt.Fprintf(buf, "%d", val.AsInt64())
		}
	case log.KindString:
		{
			_, err = buf.WriteString(val.AsString())
		}
	case log.KindBytes:
		{
			_, err = fmt.Fprintf(buf, "%x", val.AsBytes())
		}
	case log.KindSlice:
		{
			sep := func(c byte) {
				err = buf.WriteByte(c)
			}
			if sep('['); err != nil {
				return // err
			}
			for i, item := range val.AsSlice() {
				if i > 0 {
					if sep(','); err != nil {
						return // err
					}
				}
				err = enc.printValue(buf, item)
				if err != nil {
					return // err
				}
			}
			if sep(']'); err != nil {
				return // err
			}
		}
	// case log.KindMap:
	// 	{
	// 		for _, att := range value.AsMap() {
	// 			err = enc.printAttr(buf, att, group)
	// 			if err != nil {
	// 				return // err
	// 			}
	// 		}
	// 	}
	// case log.KindEmpty:
	default:
		_, err = buf.WriteString(val.AsString())
	}
	return // err?
}

func init() {
	codec.Register("text", NewCodec)
}
