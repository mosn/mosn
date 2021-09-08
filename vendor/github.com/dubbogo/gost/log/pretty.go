/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// package gxlog is based on log4go.
// pretty.go provides pretty format string
package gxlog

import (
	"github.com/davecgh/go-spew/spew"

	"github.com/k0kubun/pp"
)

func PrettyString(i interface{}) string {
	return spew.Sdump(i)
}

func ColorSprint(i interface{}) string {
	return pp.Sprint(i)
}

func ColorSprintln(i interface{}) string {
	return pp.Sprintln(i)
}

func ColorSprintf(fmt string, args ...interface{}) string {
	return pp.Sprintf(fmt, args...)
}

func ColorPrint(i interface{}) {
	pp.Print(i)
}

func ColorPrintln(i interface{}) {
	pp.Println(i)
}

func ColorPrintf(fmt string, args ...interface{}) {
	pp.Printf(fmt, args...)
}
