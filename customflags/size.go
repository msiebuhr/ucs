package customflags

import (
	"github.com/docker/go-units"
)

// Quick-and-dirty human-readable sizes
type Size int64

func (v Size) String() string {
	return units.BytesSize(float64(v))
}

func (v Size) Int64() int64 {
	return int64(v)
}

func (v *Size) Set(s string) error {
	b, err := units.RAMInBytes(s)
	if err != nil {
		return err
	}
	*v = Size(b)
	return nil
}

func NewSize(s int64) *Size {
	size := Size(s)
	return &size
}
