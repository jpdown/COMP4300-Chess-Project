package logging

import (
	"fmt"
	"os"
)

var verbose = false

func Init() {
	// Check the arguments for the -v verbose flag
	for argIndex := range os.Args {
		if os.Args[argIndex] == "-v" {
			verbose = true
		}
	}
}

func Log(message string) {
	// Analogous to println, always prints
	println(message)
}

func Logf(format string, args ...any) {
	// Analogous to printf, always prints
	fmt.Printf(format, args...)
}

func Debug(message string) {
	// Analogous to println, only prints in verbose mode
	if verbose {
		println(message)
	}
}

func Debugf(format string, args ...any) {
	// Analogous to printf, only prints in verbose mode
	if verbose {
		fmt.Printf(format, args...)
	}
}
