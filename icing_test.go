package structcheck

import (
	"github.com/stretchr/testify/require"
	"testing"
)

type pointyTestStruct struct {
	A *int
	b *pointyTestStructInner
}

type pointyTestStructInner struct {
	a *int
	b int
}

var pointyTestStructFieldNames = []string{
	"A",
	"b",
	"b.a",
	"b.b",
}

func TestCheckFieldsExist_good(t *testing.T) {
	require.NoError(t, CheckFieldsExist(pointyTestStruct{}, pointyTestStructFieldNames))
}

func TestCheckFieldsExist_bad(t *testing.T) {
	require.Error(t, CheckFieldsExist(pointyTestStruct{}, []string{"C"}))
}

func TestCheckFieldsNotNil_good(t *testing.T) {
	zero := 0
	testStruct := pointyTestStruct{&zero, &pointyTestStructInner{&zero, zero}}
	require.NoError(t, CheckFieldsNotNil(testStruct, pointyTestStructFieldNames))
}

func TestCheckFieldNotNil_badFieldNames(t *testing.T) {
	zero := 0
	testStruct := pointyTestStruct{&zero, &pointyTestStructInner{&zero, zero}}
	require.Error(t, CheckFieldsNotNil(testStruct, []string{"C"}))
}

func TestCheckFieldNotNil_badNilField(t *testing.T) {
	require.Error(t, CheckFieldsNotNil(pointyTestStruct{}, pointyTestStructFieldNames))
}

func TestCheckFieldNotNil_badNestedNilField(t *testing.T) {
	zero := 0
	testStruct := pointyTestStruct{&zero, &pointyTestStructInner{nil, zero}}
	require.Error(t, CheckFieldsNotNil(testStruct, pointyTestStructFieldNames))
}

func TestCheckFieldNotNil_badFieldNestedUnderNil(t *testing.T) {
	require.Error(t, CheckFieldsNotNil(pointyTestStruct{}, []string{"b.a"}))
}
