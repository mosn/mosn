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

package java_sql_time

import "time"

type Time struct {
	time.Time
}

func (Time) JavaClassName() string {
	return "java.sql.Time"
}

func (t *Time) GetTime() time.Time {
	return t.Time
}

// nolint
func (t *Time) Hour() int {
	return t.Time.Hour()
}

// nolint
func (t *Time) Minute() int {
	return t.Time.Minute()
}

// nolint
func (t *Time) Second() int {
	return t.Time.Second()
}

func (t *Time) SetTime(time time.Time) {
	t.Time = time
}

func (t *Time) ValueOf(timeStr string) error {
	time, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		return err
	}
	t.Time = time
	return nil
}
