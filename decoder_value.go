package kdl

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"time"
)

// unmarshalValue unmarshals a KDL value into a single Go value. See [Unmarshal]
// for details on supported value types. It handles special cases like
// [time.Time], [time.Duration], and [ValueUnmarshaler], using an optional
// format string for guidance.
func (d *decoder) unmarshalValue(value Value, tag structTag, target reflect.Value) error {
	if target.Type().NumMethod() > 0 && target.CanInterface() {
		if u, ok := target.Interface().(ValueUnmarshaler); ok {
			return u.UnmarshalKDLValue(value)
		}
	}

	switch target.Type() {
	case timeType:
		return d.unmarshalTime(value, tag, target)
	case durationType:
		return d.unmarshalDuration(value, tag, target)
	}

	switch target.Kind() {
	case reflect.String:
		return d.unmarshalString(value, tag, target)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return d.unmarshalInt(value, tag, target)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return d.unmarshalUint(value, tag, target)
	case reflect.Float32, reflect.Float64:
		return d.unmarshalFloat(value, tag, target)
	case reflect.Bool:
		return d.unmarshalBool(value, tag, target)
	case reflect.Pointer:
		elem := target.Type().Elem()
		switch elem {
		case bigIntType:
			return d.unmarshalBigInt(value, tag, target)
		case bigFloatType:
			return d.unmarshalBigFloat(value, tag, target)
		default:
			if target.IsNil() {
				target.Set(reflect.New(elem))
			}
			return d.unmarshalValue(value, tag, target.Elem())
		}
	case reflect.Interface:
		return d.unmarshalValueIntoInterface(value, target)
	}

	return nil
}

// unmarshalString unmarshals a KDL value into a Go string, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalString(v Value, tag structTag, target reflect.Value) error {
	if d.strict || tag.flags&strict != 0 {
		if v.Kind() == String {
			target.SetString(v.String())
			return nil
		}
		return fmt.Errorf("%w: cannot unmarshal %T into string", ErrStrict, v)
	}
	switch v.Kind() {
	case String:
		target.SetString(v.String())
	case Int:
		target.SetString(strconv.FormatInt(int64(v.Int()), 10))
	case Float:
		target.SetString(strconv.FormatFloat(v.Float(), 'f', -1, 64))
	case BigInt:
		target.SetString(v.BigInt().String())
	case BigFloat:
		target.SetString(v.BigFloat().String())
	case Bool:
		target.SetString(strconv.FormatBool(v.Bool()))
	case Null:
		target.SetString("")
	default:
		return fmt.Errorf("cannot unmarshal %T into string", v)
	}
	return nil
}

// unmarshalInt unmarshals a KDL value into a Go integer, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalInt(v Value, tag structTag, target reflect.Value) error {
	if d.strict || tag.flags&strict != 0 {
		switch v.Kind() {
		case Int:
			return d.setInt(target, int64(v.Int()))
		case BigInt:
			if !v.BigInt().IsInt64() {
				return fmt.Errorf("bigint value %s overflows int64", v.BigInt().String())
			}
			return d.setInt(target, v.BigInt().Int64())
		}
		return fmt.Errorf("%w: cannot unmarshal %T into int", ErrStrict, v)
	}
	switch v.Kind() {
	case Int:
		return d.setInt(target, int64(v.Int()))
	case Float:
		return d.setInt(target, int64(v.Float()))
	case BigInt:
		if !v.BigInt().IsInt64() {
			return fmt.Errorf("bigint value %s overflows int64", v.BigInt().String())
		}
		return d.setInt(target, v.BigInt().Int64())
	case BigFloat:
		i, _ := v.BigFloat().Int64()
		return d.setInt(target, i)
	case Bool:
		if v.Bool() {
			return d.setInt(target, 1)
		} else {
			return d.setInt(target, 0)
		}
	case String:
		// try to parse the string as an int
		i, err := strconv.ParseInt(v.String(), 0, 64)
		if err != nil {
			return fmt.Errorf("cannot unmarshal %q into integer: %w", v.String(), err)
		}
		return d.setInt(target, i)
	case Null:
		return d.setInt(target, 0)
	default:
		return fmt.Errorf("cannot unmarshal %T into int", v)
	}
}

func (d *decoder) setInt(target reflect.Value, value int64) error {
	if !target.CanInt() {
		panic("setInt called on non-int target")
	}
	if target.OverflowInt(value) {
		return fmt.Errorf("integer value %d overflows target type %s", value, target.Type().String())
	}
	target.SetInt(value)
	return nil
}

// unmarshalUint unmarshals a KDL value into a Go unsigned integer, converting
// as needed outside of strict mode.
func (d *decoder) unmarshalUint(v Value, tag structTag, target reflect.Value) error {
	if d.strict || tag.flags&strict != 0 {
		switch v.Kind() {
		case Int:
			if v.Int() < 0 {
				return fmt.Errorf("cannot unmarshal negative integer %d into uint", v.Int())
			}
			return d.setUint(target, uint64(v.Int()))
		case BigInt:
			if v.BigInt().Sign() < 0 {
				return fmt.Errorf("cannot unmarshal negative bigint %s into uint", v.BigInt().String())
			}
			if !v.BigInt().IsUint64() {
				return fmt.Errorf("bigint value %s overflows uint64", v.BigInt().String())
			}
			return d.setUint(target, v.BigInt().Uint64())
		}
		return fmt.Errorf("%w: cannot unmarshal %T into uint", ErrStrict, v)
	}
	switch v.Kind() {
	case Int:
		i := v.Int()
		if i < 0 {
			return fmt.Errorf("cannot unmarshal negative integer %d into uint", i)
		}
		return d.setUint(target, uint64(i))
	case Float:
		f := v.Float()
		if f < 0 {
			return fmt.Errorf("cannot unmarshal negative float %f into uint", f)
		}
		return d.setUint(target, uint64(f))
	case BigInt:
		bi := v.BigInt()
		if bi.Sign() < 0 {
			return fmt.Errorf("cannot unmarshal negative bigint %s into uint", bi.String())
		}
		if !bi.IsUint64() {
			return fmt.Errorf("bigint value %s overflows uint64", bi.String())
		}
		return d.setUint(target, bi.Uint64())
	case BigFloat:
		bf := v.BigFloat()
		i, _ := bf.Int64()
		if i < 0 {
			return fmt.Errorf("cannot unmarshal negative bigfloat %s into uint", bf.String())
		}
		return d.setUint(target, uint64(i))
	case Bool:
		if v.Bool() {
			return d.setUint(target, 1)
		} else {
			return d.setUint(target, 0)
		}
	case String:
		u, err := strconv.ParseUint(v.String(), 0, 64)
		if err != nil {
			return fmt.Errorf("cannot unmarshal %q into unsigned integer: %w", v.String(), err)
		}
		return d.setUint(target, u)
	case Null:
		return d.setUint(target, 0)
	default:
		return fmt.Errorf("cannot unmarshal %T into uint", v)
	}
}

func (d *decoder) setUint(target reflect.Value, value uint64) error {
	if !target.CanUint() {
		panic("setUint called on non-uint target")
	}
	if target.OverflowUint(value) {
		return fmt.Errorf("unsigned integer value %d overflows target type %s", value, target.Type().String())
	}
	target.SetUint(value)
	return nil
}

// unmarshalFloat unmarshals a KDL value into a Go float, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalFloat(v Value, tag structTag, target reflect.Value) error {
	if d.strict || tag.flags&strict != 0 {
		switch v.Kind() {
		case Float:
			target.SetFloat(v.Float())
			return nil
		case BigFloat:
			f64, _ := v.BigFloat().Float64()
			target.SetFloat(f64)
			return nil
		}
		return fmt.Errorf("%w: cannot unmarshal %T into float", ErrStrict, v)
	}
	switch v.Kind() {
	case Int:
		target.SetFloat(float64(v.Int()))
	case Float:
		target.SetFloat(v.Float())
	case BigInt:
		f, _ := v.BigInt().Float64()
		target.SetFloat(f)
	case BigFloat:
		f, _ := v.BigFloat().Float64()
		target.SetFloat(f)
	case Bool:
		if v.Bool() {
			target.SetFloat(1)
		} else {
			target.SetFloat(0)
		}
	case String:
		// try to parse the string as a float
		f, err := strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return fmt.Errorf("cannot unmarshal %q into float: %w", v.String(), err)
		}
		target.SetFloat(f)
	case Null:
		target.SetFloat(0)
	default:
		return fmt.Errorf("cannot unmarshal %T into float", v)
	}
	return nil
}

// unmarshalBool unmarshals a KDL value into a Go bool, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalBool(v Value, tag structTag, target reflect.Value) error {
	if d.strict || tag.flags&strict != 0 {
		if v.Kind() == Bool {
			target.SetBool(v.Bool())
			return nil
		}
		return fmt.Errorf("%w: cannot unmarshal %T into bool", ErrStrict, v)
	}
	switch v.Kind() {
	case Bool:
		target.SetBool(v.Bool())
	case Int:
		target.SetBool(v.Int() != 0)
	case Float:
		target.SetBool(v.Float() != 0)
	case BigInt:
		target.SetBool(v.BigInt().Sign() != 0)
	case BigFloat:
		i, _ := v.BigFloat().Int64()
		target.SetBool(i != 0)
	case String:
		switch v.String() {
		case "1", "t", "T", "true", "TRUE", "True", "y", "Y", "yes", "YES", "Yes":
			target.SetBool(true)
		case "0", "f", "F", "false", "FALSE", "False", "n", "N", "no", "NO", "No":
			target.SetBool(false)
		default:
			return fmt.Errorf("cannot unmarshal %q into bool", v.String())
		}
	case Null:
		target.SetBool(false)
	default:
		return fmt.Errorf("cannot unmarshal %T into bool", v)
	}
	return nil
}

// unmarshalBigInt unmarshals a KDL value into a Go big.Int, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalBigInt(v Value, tag structTag, target reflect.Value) error {
	// targetField is a pointer to a big.Int
	bi := target.Interface().(*big.Int)
	if d.strict || tag.flags&strict != 0 {
		switch v.Kind() {
		case Int:
			bi.SetInt64(int64(v.Int()))
			return nil
		case BigInt:
			bi.Set(v.BigInt())
			return nil
		}
		return fmt.Errorf("%w: cannot unmarshal %T into big.Int", ErrStrict, v)
	}
	switch v.Kind() {
	case Int:
		bi.SetInt64(int64(v.Int()))
	case Float:
		bi.SetInt64(int64(v.Float()))
	case BigInt:
		bi.Set(v.BigInt())
	case BigFloat:
		i, _ := v.BigFloat().Int64()
		bi.SetInt64(i)
	case Bool:
		if v.Bool() {
			bi.SetInt64(1)
		} else {
			bi.SetInt64(0)
		}
	case String:
		i, ok := new(big.Int).SetString(v.String(), 10)
		if !ok {
			return fmt.Errorf("cannot unmarshal %q into big.Int", v.String())
		}
		bi.Set(i)
	case Null:
		bi.SetInt64(0)
	default:
		return fmt.Errorf("cannot unmarshal %T into big.Int", v)
	}

	return nil
}

// unmarshalBigFloat unmarshals a KDL value into a Go big.Float, converting as needed
// outside of strict mode.
func (d *decoder) unmarshalBigFloat(v Value, tag structTag, target reflect.Value) error {
	// targetField is a pointer to a big.Float
	bf := target.Interface().(*big.Float)
	if d.strict || tag.flags&strict != 0 {
		switch v.Kind() {
		case Float:
			bf.SetFloat64(v.Float())
			return nil
		case BigFloat:
			bf.Set(v.BigFloat())
			return nil
		}
		return fmt.Errorf("%w: cannot unmarshal %T into big.Float", ErrStrict, v)
	}
	switch v.Kind() {
	case Int:
		bf.SetInt64(int64(v.Int()))
	case Float:
		bf.SetFloat64(v.Float())
	case BigInt:
		bf.SetInt(v.BigInt())
	case BigFloat:
		bf.Set(v.BigFloat())
	case Bool:
		if v.Bool() {
			bf.SetFloat64(1)
		} else {
			bf.SetFloat64(0)
		}
	case String:
		f, ok := new(big.Float).SetString(v.String())
		if !ok {
			return fmt.Errorf("cannot unmarshal %q into big.Float", v.String())
		}
		bf.Set(f)
	case Null:
		bf.SetFloat64(0)
	default:
		return fmt.Errorf("cannot unmarshal %T into big.Float", v)
	}
	return nil
}

// unmarshalTime unmarshals a KDL value into a Go time.Time, converting as needed.
// format is interpreted using the same rules as time.Parse, using [time.RFC3339] as
// the default if empty.
// TODO: strict mode handling
func (d *decoder) unmarshalTime(v Value, tag structTag, target reflect.Value) error {
	// target is a time.Time
	format := tag.format
	if format == "" {
		format = time.RFC3339
	}

	origFormat := format
	base, format := decodeFormat(format)
	if format == "" && base == math.MaxUint {
		return fmt.Errorf("unknown time format %q", origFormat)
	}

	var t time.Time
	var err error
	switch v.Kind() {
	case String:
		t, err = parseTime(base, format, v.String())
	case Int:
		t, err = parseTimeUnix([]byte(strconv.FormatInt(int64(v.Int()), 10)), base)
	case Float:
		t, err = parseTimeUnix([]byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), base)
	case BigInt:
		t, err = parseTimeUnix([]byte(v.BigInt().String()), base)
	case BigFloat:
		t, err = parseTimeUnix([]byte(v.BigFloat().String()), base)
	case Null:
		// zero time
	default:
		return fmt.Errorf("cannot unmarshal %T into time.Time", v)
	}

	if err != nil {
		return fmt.Errorf("cannot unmarshal %q into time.Time: %w", v, err)
	}

	target.Set(reflect.ValueOf(t))
	return nil
}

// unmarshalDuration unmarshals a KDL value into a Go [time.Duration],
// converting as needed. format must be one of units (corresponding to
// [time.ParseDuration]), sec, milli, micro, or nano.
// TODO: strict mode handling
func (d *decoder) unmarshalDuration(v Value, tag structTag, target reflect.Value) error {
	base := uint64(0)
	switch tag.format {
	case "", "units":
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
		return fmt.Errorf("unknown duration format %q", tag.format)
	}

	var td time.Duration
	var err error
	switch v.Kind() {
	case String:
		td, err = parseDuration(base, v.String())
	case Int:
		td, err = parseDurationBase10([]byte(strconv.FormatInt(int64(v.Int()), 10)), base)
	case Float:
		td, err = parseDurationBase10([]byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), base)
	case BigInt:
		td, err = parseDurationBase10([]byte(v.BigInt().String()), base)
	case BigFloat:
		td, err = parseDurationBase10([]byte(v.BigFloat().String()), base)
	case Null:
		// zero duration
	default:
		return fmt.Errorf("cannot unmarshal %T into time.Duration", v)
	}

	if err != nil {
		return fmt.Errorf("cannot unmarshal %q into time.Duration: %w", v, err)
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
	if v.Kind() == Int && v.Int() <= math.MaxInt {
		target.Set(reflect.ValueOf(v.Int()))
		return nil
	}

	if val.Type().ConvertibleTo(target.Type()) {
		target.Set(val.Convert(target.Type()))
	} else {
		return fmt.Errorf("cannot unmarshal %T into %s", v, target.Type().String())
	}
	return nil
}
