package kdl

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// unmarshalString unmarshals a KDL value into a Go string, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalString(v Value, target reflect.Value) error {
	if d.strict {
		if s, ok := v.(String); ok {
			target.SetString(s.value)
			return nil
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into string", v)
	}
	switch v := v.(type) {
	case String:
		target.SetString(v.value)
	case Integer:
		target.SetString(strconv.FormatInt(v.value, 10))
	case Float:
		target.SetString(strconv.FormatFloat(v.value, 'f', -1, 64))
	case BigInt:
		target.SetString(v.value.String())
	case BigFloat:
		target.SetString(v.value.String())
	case Boolean:
		target.SetString(strconv.FormatBool(v.value))
	case Null:
		target.SetString("")
	default:
		return errors.Errorf("cannot unmarshal %T into string", v)
	}
	return nil
}

// unmarshalInt unmarshals a KDL value into a Go integer, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalInt(v Value, target reflect.Value) error {
	if d.strict {
		if i, ok := v.(Integer); ok {
			return d.setInt(target, i.value)
		}
		if i, ok := v.(BigInt); ok {
			return d.setInt(target, i.value.Int64())
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into int", v)
	}
	switch v := v.(type) {
	case Integer:
		return d.setInt(target, v.value)
	case Float:
		return d.setInt(target, int64(v.value))
	case BigInt:
		return d.setInt(target, v.value.Int64())
	case BigFloat:
		i, _ := v.value.Int64()
		return d.setInt(target, i)
	case Boolean:
		if v.value {
			return d.setInt(target, 1)
		} else {
			return d.setInt(target, 0)
		}
	case String:
		// try to parse the string as an int
		value := v.value
		base := 10
		if len(v.value) > 2 && v.value[0] == '0' {
			switch v.value[1] {
			case 'x', 'X':
				base = 16
				value = v.value[2:]
			case 'b', 'B':
				base = 2
				value = v.value[2:]
			case 'o', 'O':
				base = 8
				value = v.value[2:]
			}
		}
		i, err := strconv.ParseInt(value, base, 64)
		if err != nil {
			return errors.Wrapf(err, "cannot unmarshal %q into integer", v.value)
		}
		fmt.Printf("case\n")
		return d.setInt(target, i)
	case Null:
		return d.setInt(target, 0)
	default:
		return errors.Errorf("cannot unmarshal %T into int", v)
	}
}

func (d *decoder) setInt(target reflect.Value, value int64) error {
	if !target.CanInt() {
		panic("setInt called on non-int target")
	}
	if target.OverflowInt(value) {
		return errors.Errorf("integer value %d overflows target type %s", value, target.Type().String())
	}
	target.SetInt(value)
	return nil
}

// unmarshalUint unmarshals a KDL value into a Go unsigned integer, converting
// as needed outside of strict mode.
func (d *decoder) unmarshalUint(v Value, target reflect.Value) error {
	if d.strict {
		if i, ok := v.(Integer); ok {
			if i.value < 0 {
				return errors.Errorf("cannot unmarshal negative integer %d into uint", i.value)
			}
			return d.setUint(target, uint64(i.value))
		}
		if i, ok := v.(BigInt); ok {
			if i.value.Sign() < 0 {
				return errors.Errorf("cannot unmarshal negative bigint %s into uint", i.value.String())
			}
			return d.setUint(target, uint64(i.value.Int64()))
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into uint", v)
	}
	switch v := v.(type) {
	case Integer:
		if v.value < 0 {
			return errors.Errorf("cannot unmarshal negative integer %d into uint", v.value)
		}
		return d.setUint(target, uint64(v.value))
	case Float:
		if v.value < 0 {
			return errors.Errorf("cannot unmarshal negative float %f into uint", v.value)
		}
		return d.setUint(target, uint64(v.value))
	case BigInt:
		if v.value.Sign() < 0 {
			return errors.Errorf("cannot unmarshal negative bigint %s into uint", v.value.String())
		}
		return d.setUint(target, uint64(v.value.Int64()))
	case BigFloat:
		i, _ := v.value.Int64()
		if i < 0 {
			return errors.Errorf("cannot unmarshal negative bigfloat %s into uint", v.value.String())
		}
		return d.setUint(target, uint64(i))
	case Boolean:
		if v.value {
			return d.setUint(target, 1)
		} else {
			return d.setUint(target, 0)
		}
	case String:
		u, err := strconv.ParseUint(v.value, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "cannot unmarshal %q into unsigned integer", v.value)
		}
		return d.setUint(target, u)
	case Null:
		return d.setUint(target, 0)
	default:
		return errors.Errorf("cannot unmarshal %T into int", v)
	}
}

func (d *decoder) setUint(target reflect.Value, value uint64) error {
	if !target.CanUint() {
		panic("setUint called on non-uint target")
	}
	if target.OverflowUint(value) {
		return errors.Errorf("unsigned integer value %d overflows target type %s", value, target.Type().String())
	}
	target.SetUint(value)
	return nil
}

// unmarshalFloat unmarshals a KDL value into a Go float, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalFloat(v Value, target reflect.Value) error {
	if d.strict {
		if f, ok := v.(Float); ok {
			target.SetFloat(f.value)
			return nil
		}
		if f, ok := v.(BigFloat); ok {
			f64, _ := f.value.Float64()
			target.SetFloat(f64)
			return nil
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into float", v)
	}
	switch v := v.(type) {
	case Integer:
		target.SetFloat(float64(v.value))
	case Float:
		target.SetFloat(v.value)
	case BigInt:
		f, _ := v.value.Float64()
		target.SetFloat(f)
	case BigFloat:
		f, _ := v.value.Float64()
		target.SetFloat(f)
	case Boolean:
		if v.value {
			target.SetFloat(1)
		} else {
			target.SetFloat(0)
		}
	case String:
		// try to parse the string as a float
		f, err := strconv.ParseFloat(v.value, 64)
		if err != nil {
			return errors.Wrapf(err, "cannot unmarshal %q into float", v.value)
		}
		target.SetFloat(f)
	case Null:
		target.SetFloat(0)
	default:
		return errors.Errorf("cannot unmarshal %T into float", v)
	}
	return nil
}

// unmarshalBool unmarshals a KDL value into a Go bool, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalBool(v Value, target reflect.Value) error {
	if d.strict {
		if b, ok := v.(Boolean); ok {
			target.SetBool(b.value)
			return nil
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into bool", v)
	}
	switch v := v.(type) {
	case Boolean:
		target.SetBool(v.value)
	case Integer:
		target.SetBool(v.value != 0)
	case Float:
		target.SetBool(v.value != 0)
	case BigInt:
		target.SetBool(v.value.Sign() != 0)
	case BigFloat:
		i, _ := v.value.Int64()
		target.SetBool(i != 0)
	case String:
		switch v.value {
		case "1", "t", "T", "true", "TRUE", "True", "y", "Y", "yes", "YES", "Yes":
			target.SetBool(true)
		case "0", "f", "F", "false", "FALSE", "False", "n", "N", "no", "NO", "No":
			target.SetBool(false)
		default:
			return errors.Errorf("cannot unmarshal %q into bool", v.value)
		}
	case Null:
		target.SetBool(false)
	default:
		return errors.Errorf("cannot unmarshal %T into bool", v)
	}
	return nil
}

// unmarshalBigInt unmarshals a KDL value into a Go big.Int, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalBigInt(v Value, target reflect.Value) error {
	// targetField is a pointer to a big.Int
	set := func(i int64) {
		target.MethodByName("SetInt64").Call([]reflect.Value{reflect.ValueOf(i)})
	}
	if d.strict {
		if i, ok := v.(Integer); ok {
			set(i.value)
			return nil
		}
		if i, ok := v.(BigInt); ok {
			target.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(i.value)})
			return nil
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into big.Int", v)
	}
	switch v := v.(type) {
	case Integer:
		set(v.value)
	case Float:
		set(int64(v.value))
	case BigInt:
		target.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(v.value)})
	case BigFloat:
		i, _ := v.value.Int64()
		set(i)
	case Boolean:
		if v.value {
			set(1)
		} else {
			set(0)
		}
	case String:
		i, ok := new(big.Int).SetString(v.value, 10)
		if !ok {
			return errors.Errorf("cannot unmarshal %q into big.Int", v.value)
		}
		target.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(i)})
	case Null:
		set(0)
	default:
		return errors.Errorf("cannot unmarshal %T into big.Int", v)
	}

	return nil
}

// unmarshalBigFloat unmarshals a KDL value into a Go big.Float, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalBigFloat(v Value, target reflect.Value) error {
	// targetField is a pointer to a big.Float
	set := func(f float64) {
		target.MethodByName("SetFloat64").Call([]reflect.Value{reflect.ValueOf(f)})
	}
	if d.strict {
		if f, ok := v.(Float); ok {
			set(f.value)
			return nil
		}
		if f, ok := v.(BigFloat); ok {
			target.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(f.value)})
			return nil
		}
		return errors.Wrapf(ErrStrict, "cannot unmarshal %T into big.Float", v)
	}
	switch v := v.(type) {
	case Integer:
		set(float64(v.value))
	case Float:
		set(v.value)
	case BigInt:
		f, _ := v.value.Float64()
		set(f)
	case BigFloat:
		target.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(v.value)})
	case Boolean:
		if v.value {
			set(1)
		} else {
			set(0)
		}
	case String:
		f, ok := new(big.Float).SetString(v.value)
		if !ok {
			return errors.Errorf("cannot unmarshal %q into big.Float", v.value)
		}
		target.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(f)})
	case Null:
		set(0)
	default:
		return errors.Errorf("cannot unmarshal %T into big.Float", v)
	}
	return nil
}

// unmarshalTime unmarshals a KDL value into a Go time.Time, converting as needed.
// format is interpreted using the same rules as time.Parse, using [time.RFC3339] as
// the default if empty.
// TODO: strict mode handling
func (d *decoder) unmarshalTime(v Value, format string, target reflect.Value) error {
	// target is a time.Time
	if format == "" {
		format = time.RFC3339
	}

	base, format := decodeFormat(format)
	if format == "" {
		return errors.Errorf("unknown time format %q", format)
	}

	var t time.Time
	var err error
	switch v := v.(type) {
	case String:
		t, err = parseTime(base, format, v.value)
	case Integer:
		t, err = parseTimeUnix([]byte(strconv.FormatInt(v.value, 10)), base)
	case Float:
		t, err = parseTimeUnix([]byte(strconv.FormatFloat(v.value, 'f', -1, 64)), base)
	case BigInt:
		t, err = parseTimeUnix([]byte(v.value.String()), base)
	case BigFloat:
		t, err = parseTimeUnix([]byte(v.value.String()), base)
	case Null:
		// zero time
	default:
		return errors.Errorf("cannot unmarshal %T into time.Time", v)
	}

	if err != nil {
		return errors.Wrapf(err, "cannot unmarshal %q into time.Time", v)
	}

	target.Set(reflect.ValueOf(t))
	return nil
}

// unmarshalDuration unmarshals a KDL value into a Go [time.Duration],
// converting as needed. format must be one of units (corresponding to
// [time.ParseDuration]), sec, milli, micro, or nano.
// TODO: strict mode handling
func (d *decoder) unmarshalDuration(v Value, format string, target reflect.Value) error {
	base := uint64(0)
	switch format {
	case "units":
		base = 0
	case "sec":
		base = 1e9
	case "milli":
		base = 1e6
	case "micro":
		base = 1e3
	case "nano":
		base = 1e0
	default:
		return errors.Errorf("unknown duration format %q", format)
	}

	var td time.Duration
	var err error
	switch v := v.(type) {
	case String:
		td, err = parseDuration(base, v.value)
	case Integer:
		td, err = parseDurationBase10([]byte(strconv.FormatInt(v.value, 10)), base)
	case Float:
		td, err = parseDurationBase10([]byte(strconv.FormatFloat(v.value, 'f', -1, 64)), base)
	case BigInt:
		td, err = parseDurationBase10([]byte(v.value.String()), base)
	case BigFloat:
		td, err = parseDurationBase10([]byte(v.value.String()), base)
	case Null:
		// zero duration
	default:
		return errors.Errorf("cannot unmarshal %T into time.Duration", v)
	}

	if err != nil {
		return errors.Wrapf(err, "cannot unmarshal %q into time.Duration", v)
	}

	target.Set(reflect.ValueOf(td))
	return err
}

// unmarshalValueIntoInterface unmarshals a KDL value into a Go interface value.
// It converts v into the underlying type of target if possible; for KDL
// integers, it prefers int over int64. If conversion is not possible, it
// returns an error.
func (d *decoder) unmarshalValueIntoInterface(v Value, target reflect.Value) error {
	val := reflect.ValueOf(v.RawValue())
	// special case: if unmarshaling an int into any, prefer int over int64
	if intVal, ok := v.(Integer); ok && intVal.value <= math.MaxInt {
		target.Set(reflect.ValueOf(int(intVal.value)))
		return nil
	}

	if val.Type().ConvertibleTo(target.Type()) {
		target.Set(val.Convert(target.Type()))
	} else {
		return errors.Errorf("cannot unmarshal %T into %s", v, target.Type().String())
	}
	return nil
}
