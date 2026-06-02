package kdl

import (
	"fmt"
	"math"
	"time"
)

// formatTimeValue converts a time.Time into a KDL [Value] using the given
// format string.
//
//   - the empty string defaults to RFC3339;
//   - "unix", "unixmilli", "unixmicro", and "unixnano" emit integer values;
//   - all other strings are passed to [time.Time.Format] and emitted as strings.
func formatTimeValue(t time.Time, format string) (Value, error) {
	if format == "" {
		format = "RFC3339"
	}
	base, layout := decodeFormat(format)
	if layout == "" && base == math.MaxUint {
		return Value{}, fmt.Errorf("unknown time format %q", format)
	}
	switch base {
	case 0, math.MaxUint: // 0 is RFC3339 or RFC3339Nano, math.MaxUint is all other named/custom formats
		return NewString(t.Format(layout)), nil
	case 1e0:
		return NewInt(int(t.Unix())), nil
	case 1e3:
		return NewInt(int(t.UnixMilli())), nil
	case 1e6:
		return NewInt(int(t.UnixMicro())), nil
	case 1e9:
		return NewInt(int(t.UnixNano())), nil
	default:
		// decodeFormat returned an unsupported base?
		panic(fmt.Sprintf("kdl.Encode: formatTimeValue invalid time format base %d", base))
	}
}

// formatDurationValue converts a time.Duration into a KDL [Value] using the
// given format string.
//
//   - "" or "units" emits a string in [time.Duration.String] form (e.g. "1h30m");
//   - "sec", "milli", "micro", and "nano" emit a numeric value scaled accordingly,
//     using an integer when the duration divides evenly and a float otherwise.
func formatDurationValue(d time.Duration, format string) (Value, error) {
	if format == "" {
		format = "units"
	}
	var divisor int64
	switch format {
	case "units":
		return NewString(d.String()), nil
	case "sec":
		divisor = int64(time.Second)
	case "milli":
		divisor = int64(time.Millisecond)
	case "micro":
		divisor = int64(time.Microsecond)
	case "nano":
		divisor = int64(time.Nanosecond)
	default:
		return Value{}, fmt.Errorf("unknown duration format %q", format)
	}
	n := int64(d)
	if n%divisor == 0 {
		return NewInt(int(n / divisor)), nil
	}
	return NewFloat(float64(n) / float64(divisor)), nil
}
