package structcheck

import (
	"fmt"
	"reflect"
)

type Check string

const (
	CheckNotNil    Check = "NotNil"
	CheckNil             = "Nil"
	CheckPositive        = "Positive"
	CheckNegative        = "Negative"
	CheckNoSign          = "NoSign"    // -0 is considered signless
	CheckNotEmpty        = "NotEmpty"  // len(o) != 0
	CheckEmpty           = "Empty"     // len(o) == 0
	CheckNillable        = "Nillable"  // can be nil
	CheckNumeric         = "Numeric"   // is a number (includes complex)
	CheckContainer       = "Container" // is a valid param for len()
)

var str2check = map[string]Check{}

func init() {
	for _, check := range []Check{
		CheckNotNil,
		CheckNil,
		CheckPositive,
		CheckNegative,
		CheckNoSign,
		CheckNotEmpty,
		CheckNillable,
		CheckNumeric,
		CheckContainer,
	} {
		str2check[string(check)] = check
	}
}

type kindClass map[reflect.Kind]interface{}

func (class kindClass) Check(v metaValue) bool {
	_, ok := class[v.Kind()]
	return ok
}

var nillable = kindClass{
	reflect.Ptr:       nil,
	reflect.Interface: nil,
	reflect.Chan:      nil,
	reflect.Func:      nil,
	reflect.Map:       nil,
	reflect.Slice:     nil,
}

var numeric = kindClass{
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

var container = kindClass{
	reflect.String: nil,
	reflect.Array:  nil,
	reflect.Slice:  nil,
	reflect.Map:    nil,
	reflect.Chan:   nil,
}

type sign int

const (
	signNone sign = iota
	signPositive
	signNegative
)

func checkSign(v metaValue, s sign) bool {
	if !numeric.Check(v) {
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
	case signNone:
		checkPassed = !isNegative && !isPositive
	case signNegative:
		checkPassed = isNegative
	case signPositive:
		checkPassed = isPositive
	default:
		panic(fmt.Sprintf("unrecognized sign: %v", s))
	}
	return checkPassed
}

type checker func(v metaValue) bool

var check2checker = map[Check]checker{
	CheckNotNil:    func(v metaValue) bool { return !(nillable.Check(v) && v.IsNil()) },
	CheckNil:       func(v metaValue) bool { return !(nillable.Check(v) && !v.IsNil()) },
	CheckPositive:  func(v metaValue) bool { return checkSign(v, signPositive) },
	CheckNegative:  func(v metaValue) bool { return checkSign(v, signNegative) },
	CheckNoSign:    func(v metaValue) bool { return checkSign(v, signNone) },
	CheckNotEmpty:  func(v metaValue) bool { return !(container.Check(v) && v.Len() == 0) },
	CheckEmpty:     func(v metaValue) bool { return !(container.Check(v) && v.Len() != 0) },
	CheckNillable:  func(v metaValue) bool { return nillable.Check(v) },
	CheckNumeric:   func(v metaValue) bool { return numeric.Check(v) },
	CheckContainer: func(v metaValue) bool { return container.Check(v) },
}
