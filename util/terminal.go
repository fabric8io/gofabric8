package util

import (
	"fmt"
	"os"

	"github.com/daviddengcn/go-colortext"
)

func Infof(msg string, args ...interface{}) {
	Info(fmt.Sprintf(msg, args...))
}

func Info(msg string) {
	ct.ChangeColor(ct.Blue, false, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func Blank() {
	fmt.Println()
}

func Fatalf(msg string, args ...interface{}) {
	Fatal(fmt.Sprintf(msg, args...))
}

func Fatal(msg string) {
	ct.ChangeColor(ct.Red, true, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
	os.Exit(1)
}

func Success(msg string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func Successf(msg string, args ...interface{}) {
	Success(fmt.Sprintf(msg, args...))
}

func Failure(msg string) {
	ct.ChangeColor(ct.Red, false, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func Failuref(msg string, args ...interface{}) {
	Failure(fmt.Sprintf(msg, args...))
}
