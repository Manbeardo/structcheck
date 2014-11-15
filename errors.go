package structcheck

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"
)

// returned when the top level object does not drill down to a struct.
type ErrorInvalidKind struct {
	reflect.Kind
}

func (e ErrorInvalidKind) Error() string {
	return fmt.Sprintf("Provided object must drill down to a struct. Received: %v", e.Kind)
}

// returned when an illegal check is encountered
type ErrorIllegalCheck struct {
	Value  metaValue
	Check  Check
	Reason string
}

func (e ErrorIllegalCheck) Error() string {
	return fmt.Sprintf("Encountered illegal check on %v: %v Reason: %v", strings.Join(e.Value.Name, "."), string(e.Check), e.Reason)
}

// returned when a top level nil is received
type ErrorNilValue struct{}

func (e ErrorNilValue) Error() string {
	return fmt.Sprintf("Provided object must drill down to a struct. Encountered nil.")
}

// returned when checks fail on fields
type ErrorChecksFailed struct {
	Field2Checks map[Field][]Check
}

func (e ErrorChecksFailed) Error() string {
	buf := new(bytes.Buffer)
	sortedFields := make([]Field, 0, len(e.Field2Checks))
	for field, _ := range e.Field2Checks {
		sortedFields = append(sortedFields, field)
	}
	sort.Sort(ByFieldOrder(sortedFields))
	failWriter := tabwriter.NewWriter(buf, 1, 4, 1, ' ', 0)
	for _, field := range sortedFields {
		checks := e.Field2Checks[field]
		fails := make([]string, 0, len(checks))
		for _, check := range checks {
			fails = append(fails, string(check))
		}
		failWriter.Write([]byte(fmt.Sprintf("\n\t%v:\t%v:\t%v", field.Name, strings.Join(fails, ", "), field.Value)))
	}
	failWriter.Flush()
	return fmt.Sprintf("The following field(s) failed checks: %v", buf.String())
}

// returned by checkers when check fails
type errorCheckFailed struct{}

func (e errorCheckFailed) Error() string { return "" }
