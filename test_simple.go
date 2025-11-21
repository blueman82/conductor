package main

import "fmt"

// Add creates a simple addition function
func Add(a, b int) int {
	return a + b
}

func main() {
	result := Add(2, 3)
	fmt.Printf("Result: %d\n", result)
}
