package ratelimit

import (
	"fmt"
	"strings"
)

// ByteUnit represents size in byte(s) count
type ByteUnit uint64

const (
	Byte     ByteUnit = 1
	Kilobyte          = (Byte << 10)
	Megabyte          = (Kilobyte << 10)
	Gigabyte          = (Megabyte << 10)

	Terabyte = (Gigabyte << 10)
	Petabyte = (Terabyte << 10)
	Exabyte  = (Petabyte << 10)

	// Zettabyte  = (Exabyte << 10)
	// Yottabyte  = (Zettabyte << 10)

	MaxSize = 1<<64 - 1 // 16Xb
)

func (v *ByteUnit) Capacity(unit ByteUnit) uint64 {
	if v != nil {
		return uint64(*v / max(unit, Byte))
	}
	return 0
}

func (v *ByteUnit) MarshalText() ([]byte, error) {
	// NULL ?
	if v == nil || *v == 0 {
		return nil, nil
	}
	text := FormatSize((*v), 1)
	return []byte(text), nil
}

var ErrParseSize = fmt.Errorf("size: invalid spec")

func (v *ByteUnit) UnmarshalText(text []byte) error {
	// NULL ?
	if len(text) == 0 {
		(*v) = 0
		return nil
	}
	setv, ok := ParseSize(string(text))
	if !ok {
		return ErrParseSize
	}
	(*v) = setv
	return nil
}

func ParseSize(spec string) (size ByteUnit, ok bool) {
	var (
		num  uint64
		unit string
	)
	n, err := fmt.Sscanf(spec, "%d%s", &num, &unit)
	if err != nil {
		// invalid rate specification
		return // Rate{}, false
	}
	_ = n // expect: >= 2
	size, ok = ParseByteUnit(unit)
	if !ok {
		// invalid time unit interval
		return // Rate{}, false
	}
	size *= ByteUnit(num)
	return size, true
}

func FormatSize(size ByteUnit, prec int) string {
	if size == 0 {
		return "0b"
	}
	var unit = Byte // current unit precision
	for unit < Exabyte && (unit<<10) <= size {
		// find suitable unit divider ..
		unit = (unit << 10)
	}

	// MaxUint64 = "16Xb"
	var buf [32]byte
	w := len(buf)
	s := byteUnitString(unit)
	w -= len(s)
	copy(buf[w:], s)

	form := float64(size) / float64(unit)
	for range prec {
		form *= 10
	}

	digit := uint64(form)
	w, digit = fmtFrac(buf[:w], digit, prec)
	// u is now integer size
	w = fmtInt(buf[:w], digit)
	return string(buf[w:])
}

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros. It omits the decimal
// point too when the fraction is 0. It returns the index where the
// output bytes begin and the value v/10**prec.
func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	// Omit trailing zeros up to and including decimal point.
	w := len(buf)
	print := false
	for range prec {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

func ParseByteUnit(unit string) (ByteUnit, bool) {
	unit = strings.ToLower(unit)
	if size, ok := unitByteMap[unit]; ok {
		return size, true
	}
	return 0, false
}

func byteUnitString(size ByteUnit) string {
	if unit, ok := byteUnitMap[size]; ok {
		return unit
	}
	return "b"
}

var (
	unitByteMap = map[string]ByteUnit{

		"b": Byte,

		"kb": Kilobyte,
		"mb": Megabyte,
		"gb": Gigabyte,

		"tb": Terabyte,
		"pb": Petabyte,
		"xb": Exabyte,
	}

	byteUnitMap = map[ByteUnit]string{

		Byte: "b",

		Kilobyte: "Kb",
		Megabyte: "Mb",
		Gigabyte: "Gb",

		Terabyte: "Tb",
		Petabyte: "Pb",
		Exabyte:  "Xb",
	}
)
