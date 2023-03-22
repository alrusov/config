package config

import (
	"time"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	Duration time.Duration
)

//----------------------------------------------------------------------------------------------------------------------------//

// UnmarshalText implements encoding.TextUnmarshaler
func (d *Duration) UnmarshalText(data []byte) error {
	duration, err := misc.Interval2Duration(string(data))
	if err == nil {
		*d = Duration(duration)
	}
	return err
}

// MarshalText implements encoding.TextMarshaler
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(misc.Int2Interval(int64(d))), nil
}

//----------------------------------------------------------------------------------------------------------------------------//

func (d Duration) D() time.Duration {
	return time.Duration(d)
}

//----------------------------------------------------------------------------------------------------------------------------//
