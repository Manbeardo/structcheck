package structcheck

import (
	"fmt"
	"reflect"
)

type Check func(v reflect.Value) bool

var Checks = map[string]Check{
	"NotNil":    func(v reflect.Value) bool { return !(Nillable.Check(v) && v.IsNil()) },
	"Nil":       func(v reflect.Value) bool { return !(Nillable.Check(v) && !v.IsNil()) },
	"Positive":  func(v reflect.Value) bool { return CheckSign(v, SignPositive) },
	"Negative":  func(v reflect.Value) bool { return CheckSign(v, SignNegative) },
	"NoSign":    func(v reflect.Value) bool { return CheckSign(v, SignNone) },
	"NotEmpty":  func(v reflect.Value) bool { return !(Container.Check(v) && v.Len() == 0) },
	"Empty":     func(v reflect.Value) bool { return !(Container.Check(v) && v.Len() != 0) },
	"Nillable":  func(v reflect.Value) bool { return Nillable.Check(v) },
	"Numeric":   func(v reflect.Value) bool { return Numeric.Check(v) },
	"Container": func(v reflect.Value) bool { return Container.Check(v) },
}

type KindClass map[reflect.Kind]interface{}

func (class KindClass) Check(v reflect.Value) bool {
	_, ok := class[v.Kind()]
	return ok
}

var Nillable = KindClass{
	reflect.Ptr:       nil,
	reflect.Interface: nil,
	reflect.Chan:      nil,
	reflect.Func:      nil,
	reflect.Map:       nil,
	reflect.Slice:     nil,
}

var Numeric = KindClass{
	reflect.Int:        nil,
	reflect.Int8:       nil,
	reflect.Int16:      nil,
	reflect.Int32:      nil,
	reflect.Int64:      nil,
	reflect.Uint:       nil,
	reflect.Uint8:      nil,
	reflect.Uint16:     nil,
	reflect.Uint32:     nil,
	reflect.Uint64:     nil,
	reflect.Float32:    nil,
	reflect.Float64:    nil,
	reflect.Complex64:  nil,
	reflect.Complex128: nil,
}

var Container = KindClass{
	reflect.String: nil,
	reflect.Array:  nil,
	reflect.Slice:  nil,
	reflect.Map:    nil,
	reflect.Chan:   nil,
}

type Sign int

const (
	SignNone Sign = iota
	SignPositive
	SignNegative
)

func CheckSign(v reflect.Value, s Sign) bool {
	if !Numeric.Check(v) {
		return true
	}
	isPositive := false
	isNegative := false
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := v.Uint()
		isPositive = u > 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		if real(c) != 0 {
			isPositive = real(c) > 0
			isNegative = real(c) < 0
		} else if imag(c) != 0 {
			isPositive = imag(c) > 0
			isNegative = imag(c) < 0
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Int()
		isPositive = i > 0
		isNegative = i < 0
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		isPositive = f > 0
		isNegative = f < 0
	}
	if isPositive && isNegative {
		panic(fmt.Sprintf("%v is both negative and positive!? very bad.", v.Interface()))
	}

	checkPassed := false
	switch s {
	case SignNone:
		checkPassed = !isNegative && !isPositive
	case SignNegative:
		checkPassed = isNegative
	case SignPositive:
		checkPassed = isPositive
	default:
		panic(fmt.Sprintf("unrecognized sign: %v", s))
	}
	return checkPassed
}
