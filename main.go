package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "this is not a useful program")
	os.Exit(1)
}
