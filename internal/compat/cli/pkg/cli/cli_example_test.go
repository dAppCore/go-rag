package cli

import core "dappco.re/go"

func ExampleExactArgs() {
	validator := ExactArgs(1)
	err := validator(&Command{}, []string{"docs"})
	core.Println(err == nil)
	// Output: true
}

func ExampleMaximumNArgs() {
	validator := MaximumNArgs(2)
	err := validator(&Command{}, []string{"docs"})
	core.Println(err == nil)
	// Output: true
}

func ExampleNewGroup() {
	cmd := NewGroup("rag", "Search docs", "Long help")
	core.Println(cmd.Use, cmd.Short, cmd.Long)
	// Output: rag Search docs Long help
}

func ExampleStyle_Render() {
	style := Style{}
	core.Println(style.Render("Collections"))
	// Output: Collections
}
