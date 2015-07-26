package main

import (
	"fmt"
	"os"

	"github.com/daviddengcn/go-colortext"
)

func infof(msg string, args ...interface{}) {
	info(fmt.Sprintf(msg, args...))
}

func info(msg string) {
	ct.Foreground(ct.Blue, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func blank() {
	fmt.Println()
}

func fatalf(msg string, args ...interface{}) {
	fatal(fmt.Sprintf(msg, args...))
}

func fatal(msg string) {
	ct.Foreground(ct.Red, true)
	fmt.Print(msg)
	os.Exit(1)
}

func success(msg string) {
	ct.Foreground(ct.Green, false)
	fmt.Print(msg)
}

func successf(msg string, args ...interface{}) {
	success(fmt.Sprintf(msg, args...))
}

func failure(msg string) {
	ct.Foreground(ct.Red, false)
	fmt.Print(msg)
}

func failuref(msg string, args ...interface{}) {
	failure(fmt.Sprintf(msg, args...))
}
