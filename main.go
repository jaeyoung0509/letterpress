package main

import (
	"fmt"
	"os"

	letterpresscmd "github.com/jaeyoung0509/letterpress/cmd/letterpress"
)

func main() {
	if err := letterpresscmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
