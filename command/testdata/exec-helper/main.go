package main

import (
	"fmt"
	"os"
	"strconv"
)

var label = "unknown"

func main() {
	fmt.Printf("LABEL=%s\n", label)
	fmt.Printf("JAVA_HOME=%s\n", os.Getenv("JAVA_HOME"))
	fmt.Printf("PATH=%s\n", os.Getenv("PATH"))
	fmt.Printf("PWD=%s\n", mustGetwd())
	fmt.Printf("ARGS=%q\n", os.Args[1:])

	if value := os.Getenv("JAVM_TEST_EXIT"); value != "" {
		code, err := strconv.Atoi(value)
		if err == nil {
			os.Exit(code)
		}
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "<error>"
	}
	return wd
}
