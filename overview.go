/*
Checks that constraints on structs are met. Constraints are read as a comma-delimited list on the 'checks' tag. Validate constraints by running `structcheck.Validate()`.

See structcheck.Check for the full list of checks.

Example:
    package main

    import (
        "bytes"
        "encoding/json"
        "fmt"
        "github.com/Manbeardo/structcheck"
    )

    type MyJsonObjectType struct {
        NestedObject struct{
            Number  int  `checks:"Positive"`
            Pointer *int `checks:"NotNil"`
        }
    }

    var badJson = []byte(`{"NestedObject":{"Number":-1}}`)

    func main() {
        var o MyJsonObjectType
        json.NewDecoder(bytes.NewBuffer(badJson)).Decode(&o)
        err := structcheck.Validate(o)
        if err != nil {
            fmt.Println(err.Error())
        }
    }
Prints:
    The following field(s) failed checks:
        MyJsonObjectType.NestedObject.Number:  Positive: (int)(-1)
        MyJsonObjectType.NestedObject.Pointer: NotNil:   (*int)(nil)
*/
package structcheck
