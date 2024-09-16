package main

import (
	"fmt"
	"os"
)

func main() { // want "main function should not contain os.Exit call"
	fmt.Println("Hello world!")

	os.Exit(0)
}
