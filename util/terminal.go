/**
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
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
	fmt.Print(msg)
}

func Blank() {
	fmt.Println()
}

func Warnf(msg string, args ...interface{}) {
	Warn(fmt.Sprintf(msg, args...))
}

func Warn(msg string) {
	ct.ChangeColor(ct.Yellow, false, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
}

func Errorf(msg string, args ...interface{}) {
	Error(fmt.Sprintf(msg, args...))
}

func Error(msg string) {
	ct.ChangeColor(ct.Red, true, ct.None, false)
	fmt.Print(msg)
	ct.ResetColor()
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
