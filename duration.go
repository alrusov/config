package config

import (
	"time"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler
func (d *Duration) UnmarshalText(data []byte) error {
	duration, err := misc.Interval2Duration(string(data)) //time.ParseDuration(string(data))
	if err == nil {
		*d = Duration(duration)
	}
	return err
}

// MarshalText implements encoding.TextMarshaler
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

//----------------------------------------------------------------------------------------------------------------------------//
