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
	ct.ChangeColor(ct.Blue, false, ct.None, false)
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
	ct.ChangeColor(ct.Red, true, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
	os.Exit(1)
}

func success(msg string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func successf(msg string, args ...interface{}) {
	success(fmt.Sprintf(msg, args...))
}

func failure(msg string) {
	ct.ChangeColor(ct.Red, false, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func failuref(msg string, args ...interface{}) {
	failure(fmt.Sprintf(msg, args...))
}
