package local

import (
	"reflect"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
	"golang.org/x/time/rate"
)

var (
	// tokenBucketUnit = sizeof(reflect.TypeFor[tokenBucket]())
	// fixedWindowUnit = sizeof(reflect.TypeFor[fixedWindow]())

	// size=96
	unitTokenBucket = ratelimit.ByteUnit(
		reflect.TypeFor[tokenBucket]().Size() +
			reflect.TypeFor[rate.Limiter]().Size() - reflect.TypeFor[uintptr]().Size(),
	)
	// size=56
	unitFixedWindow = ratelimit.ByteUnit(
		reflect.TypeFor[fixedWindow]().Size(),
	)
)

func sizeof(rtyp reflect.Type) (bytes ratelimit.ByteUnit) {
	switch rtyp.Kind() {
	case reflect.Pointer:
		{
			bytes += sizeof(rtyp.Elem())
		}
	case reflect.Struct:
		{
			for i, n := 0, rtyp.NumField(); i < n; i++ {
				// NOTE: time.Time.(*Location) load once !
				bytes += sizeof(rtyp.Field(i).Type)
			}
		}
	default:
		{
			bytes = ratelimit.ByteUnit(rtyp.Size())
		}
	}
	return // bytes
}
