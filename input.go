package main

import (
	"bufio"
	"os"
)

func InputThread(result chan<- string) {
	reader := bufio.NewReader(os.Stdin)

	// Loop until the program is terminated
	for true {
		input, err := reader.ReadString('\n')
		if err == nil {
			result <- input
		}
	}
}
