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

package log

import (
	"testing"
)

func TestParseRoller(t *testing.T) {
	errorPraseArgs := "size=100 age=10 keep=10 compress=1"
	roller, err := ParseRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	errorPraseArgs = "size = 100 age = 10 keep = 10"
	roller, err = ParseRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	errorPraseArgs = "size=100, age=10, keep=10, compress=off"
	roller, err = ParseRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}


	praseArgs := "size=100 age=10 keep=10 compress=on"
	roller, err = ParseRoller(praseArgs)
	if roller == nil {
		t.Errorf("ParseRoller failed")
	}
	if roller.MaxSize != 100 || roller.MaxAge != 10 || roller.MaxBackups != 10 || roller.Compress != true {
		t.Errorf("ParseRoller failed")
	}

	praseArgs = "size=100"
	roller, err = ParseRoller(praseArgs)
	if roller == nil {
		t.Errorf("ParseRoller failed")
	}
	if roller.MaxSize != 100 || roller.Compress != false {
		t.Errorf("ParseRoller failed")
	}

	errorPraseArgs = "A=3"
	err = InitDefaultRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	praseArgs = "size=100"
	err = InitDefaultRoller(praseArgs)
	if err != nil {
		t.Errorf("ParseRoller failed")
	}
	if defaultRoller.MaxSize != 100 || defaultRoller.Compress != false {
		t.Errorf("ParseRoller failed")
	}

	errorPraseArgs = "time=12"
	err = InitDefaultRoller(errorPraseArgs)
	if err != nil {
		t.Errorf("ParseRoller should be failed")
	}

	if defaultRoller.MaxTime != 12 {
		t.Errorf("ParseRoller failed")
	}
}
