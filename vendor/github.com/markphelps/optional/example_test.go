package optional_test

import (
	"encoding/json"
	"fmt"

	"github.com/markphelps/optional"
)

func Example_get() {
	values := []optional.String{
		optional.NewString("foo"),
		optional.NewString(""),
		optional.NewString("bar"),
		{},
	}

	for _, v := range values {
		if v.Present() {
			value, err := v.Get()
			if err == nil {
				fmt.Println(value)
			}
		}
	}

	// Output:
	// foo
	//
	// bar
}

func Example_orElse() {
	values := []optional.String{
		optional.NewString("foo"),
		optional.NewString(""),
		optional.NewString("bar"),
		{},
	}

	for _, v := range values {
		fmt.Println(v.OrElse("not present"))
	}

	// Output:
	// foo
	//
	// bar
	// not present
}

func Example_if() {
	values := []optional.String{
		optional.NewString("foo"),
		optional.NewString(""),
		optional.NewString("bar"),
		{},
	}

	for _, v := range values {
		v.If(func(s string) {
			fmt.Println("present")
		})
	}

	// Output:
	// present
	// present
	// present
}

func Example_marshalJSON() {
	type example struct {
		Field *optional.String `json:"field,omitempty"`
	}

	var values = []optional.String{
		optional.NewString("foo"),
		optional.NewString(""),
		optional.NewString("bar"),
	}

	for _, v := range values {
		out, _ := json.Marshal(&example{
			Field: &v,
		})
		fmt.Println(string(out))
	}

	// Output:
	// {"field":"foo"}
	// {"field":""}
	// {"field":"bar"}
}

func Example_unmarshalJSON() {
	var values = []string{
		`{"field":"foo"}`,
		`{"field":""}`,
		`{"field":"bar"}`,
		"{}",
	}

	for _, v := range values {
		var o = &struct {
			Field optional.String `json:"field,omitempty"`
		}{}

		if err := json.Unmarshal([]byte(v), o); err != nil {
			fmt.Println(err)
		}

		o.Field.If(func(s string) {
			fmt.Println(s)
		})
	}

	// Output:
	// foo
	//
	// bar
}
