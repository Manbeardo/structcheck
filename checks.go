package structcheck

import (
	"fmt"
	"reflect"
)

type Check string

const (
	CheckNotNil   Check = "NotNil"
	CheckNil            = "Nil"
	CheckPositive       = "Positive"
	CheckNegative       = "Negative"
	CheckNoSign         = "NoSign"
	CheckNotEmpty       = "NotEmpty"
	CheckEmpty          = "Empty"
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
	} {
		str2check[string(check)] = check
	}
}

type kindClass struct {
	Members    map[reflect.Kind]interface{}
	ReasonFunc func(v metaValue) string
}

func (class kindClass) Check(v metaValue, c Check) error {
	if _, ok := class.Members[v.Kind()]; !ok {
		return ErrorIllegalCheck{
			value:  v,
			Check:  c,
			Reason: class.ReasonFunc(v),
		}
	} else {
		return nil
	}
}

var nillable = kindClass{
	Members: map[reflect.Kind]interface{}{
		reflect.Ptr:       nil,
		reflect.Interface: nil,
		reflect.Chan:      nil,
		reflect.Func:      nil,
		reflect.Map:       nil,
		reflect.Slice:     nil,
	},
	ReasonFunc: func(v metaValue) string { return fmt.Sprintf("%v is not a nillable type", v.Type()) },
}

var numeric = kindClass{
	Members: map[reflect.Kind]interface{}{
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
	},
	ReasonFunc: func(v metaValue) string { return fmt.Sprintf("%v is not a numeric type", v.Type()) },
}

var container = kindClass{
	Members: map[reflect.Kind]interface{}{
		reflect.String: nil,
		reflect.Array:  nil,
		reflect.Slice:  nil,
		reflect.Map:    nil,
		reflect.Chan:   nil,
	},
	ReasonFunc: func(v metaValue) string { return fmt.Sprintf("%v is not a container type", v.Type()) },
}

type sign int

const (
	signNone sign = iota
	signPositive
	signNegative
)

func checkSign(v metaValue, c Check, s sign) error {
	if err := numeric.Check(v, CheckPositive); err != nil {
		return err
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

	checkFailed := false
	switch s {
	case signNone:
		checkFailed = isNegative || isPositive
	case signNegative:
		checkFailed = !isNegative
	case signPositive:
		checkFailed = !isPositive
	default:
		panic(fmt.Sprintf("unrecognized sign: %v", s))
	}
	if checkFailed {
		return errorCheckFailed{}
	} else {
		return nil
	}
}

type checker func(v metaValue) error

var check2checker = map[Check]checker{
	CheckNotNil: func(v metaValue) error {
		if err := nillable.Check(v, CheckNotNil); err != nil {
			return err
		} else if v.IsNil() {
			return errorCheckFailed{}
		} else {
			return nil
		}
	},
	CheckNil: func(v metaValue) error {
		if err := nillable.Check(v, CheckNil); err != nil {
			return err
		} else if !v.IsNil() {
			return errorCheckFailed{}
		} else {
			return nil
		}
	},
	CheckPositive: func(v metaValue) error { return checkSign(v, CheckPositive, signPositive) },
	CheckNegative: func(v metaValue) error { return checkSign(v, CheckNegative, signNegative) },
	CheckNoSign:   func(v metaValue) error { return checkSign(v, CheckNoSign, signNone) },
	CheckNotEmpty: func(v metaValue) error {
		if err := container.Check(v, CheckNotEmpty); err != nil {
			return err
		} else if v.Len() == 0 {
			return errorCheckFailed{}
		} else {
			return nil
		}
	},
	CheckEmpty: func(v metaValue) error {
		if err := container.Check(v, CheckEmpty); err != nil {
			return err
		} else if v.Len() != 0 {
			return errorCheckFailed{}
		} else {
			return nil
		}
	},
}
