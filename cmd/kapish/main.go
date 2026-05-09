package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("kapish dev")
		return
	}
	fmt.Fprintln(os.Stderr, "kapish: cobra wiring lands in Task 2")
	os.Exit(2)
}
