package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var label = "unknown"

func main() {
	fmt.Printf("LABEL=%s\n", label)
	fmt.Printf("JAVA_HOME=%s\n", os.Getenv("JAVA_HOME"))
	fmt.Printf("PATH=%s\n", os.Getenv("PATH"))
	fmt.Printf("PWD=%s\n", mustGetwd())
	fmt.Printf("ARGS=%q\n", os.Args[1:])

	// Kill this process with SIGTERM, like `kill -TERM $$` in a shell.
	if os.Getenv("JAVM_TEST_KILL_SELF") != "" {
		self, _ := os.FindProcess(os.Getpid())
		_ = self.Signal(syscall.SIGTERM)
		time.Sleep(10 * time.Second)
	}

	// Handle SIGINT like a build tool that cleans up: write the PID to the
	// ready file, wait for SIGINT and exit with 5.
	if ready := os.Getenv("JAVM_TEST_READY_FILE"); ready != "" {
		interrupts := make(chan os.Signal, 1)
		signal.Notify(interrupts, os.Interrupt)
		if err := os.WriteFile(ready, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(70)
		}
		select {
		case <-interrupts:
			fmt.Println("child cleanup")
			os.Exit(5)
		case <-time.After(10 * time.Second):
			os.Exit(71)
		}
	}

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
