package main

import (
	"fmt"

	"github.com/boolean-maybe/navidown/internal/glamour"
)

func main() {
	in := `# Hello World

This is a simple example of Markdown rendering with Glamour!
Check out the [other examples](https://github.com/boolean-maybe/navidown/internal/glamour/tree/master/examples) too.

Bye!
`

	out, _ := glamour.Render(in, "dark")
	fmt.Print(out)
}
