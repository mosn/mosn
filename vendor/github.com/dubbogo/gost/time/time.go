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

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"strconv"
	"time"
)

func TimeDayDuration(day float64) time.Duration {
	return time.Duration(day * 24 * float64(time.Hour))
}

func TimeHourDuration(hour float64) time.Duration {
	return time.Duration(hour * float64(time.Hour))
}

func TimeMinuteDuration(minute float64) time.Duration {
	return time.Duration(minute * float64(time.Minute))
}

func TimeSecondDuration(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func TimeMillisecondDuration(m float64) time.Duration {
	return time.Duration(m * float64(time.Millisecond))
}

func TimeMicrosecondDuration(m float64) time.Duration {
	return time.Duration(m * float64(time.Microsecond))
}

func TimeNanosecondDuration(n float64) time.Duration {
	return time.Duration(n * float64(time.Nanosecond))
}

// desc: convert year-month-day-hour-minute-second to int in second
// @month: 1 ~ 12
// @hour:  0 ~ 23
// @minute: 0 ~ 59
func YMD(year int, month int, day int, hour int, minute int, sec int) int {
	return int(time.Date(year, time.Month(month), day, hour, minute, sec, 0, time.Local).Unix())
}

// @YMD in UTC timezone
func YMDUTC(year int, month int, day int, hour int, minute int, sec int) int {
	return int(time.Date(year, time.Month(month), day, hour, minute, sec, 0, time.UTC).Unix())
}

func YMDPrint(sec int, nsec int) string {
	return time.Unix(int64(sec), int64(nsec)).Format("2006-01-02 15:04:05.99999")
}

func Future(sec int, f func()) {
	time.AfterFunc(TimeSecondDuration(float64(sec)), f)
}

func Unix2Time(unix int64) time.Time {
	return time.Unix(unix, 0)
}

func UnixNano2Time(nano int64) time.Time {
	return time.Unix(nano/1e9, nano%1e9)
}

func UnixString2Time(unix string) time.Time {
	i, err := strconv.ParseInt(unix, 10, 64)
	if err != nil {
		panic(err)
	}

	return time.Unix(i, 0)
}

// 注意把time转换成unix的时候有精度损失，只返回了秒值，没有用到纳秒值
func Time2Unix(t time.Time) int64 {
	return t.Unix()
}

func Time2UnixNano(t time.Time) int64 {
	return t.UnixNano()
}

func GetEndTime(format string) time.Time {
	timeNow := time.Now()
	switch format {
	case "day":
		year, month, _ := timeNow.Date()
		nextDay := timeNow.AddDate(0, 0, 1).Day()
		t := time.Date(year, month, nextDay, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)

	case "week":
		year, month, _ := timeNow.Date()
		weekday := int(timeNow.Weekday())
		weekendday := timeNow.AddDate(0, 0, 8-weekday).Day()
		t := time.Date(year, month, weekendday, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)

	case "month":
		year := timeNow.Year()
		nextMonth := timeNow.AddDate(0, 1, 0).Month()
		t := time.Date(year, nextMonth, 1, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)

	case "year":
		nextYear := timeNow.AddDate(1, 0, 0).Year()
		t := time.Date(nextYear, 1, 1, 0, 0, 0, 0, time.Local)
		return time.Unix(t.Unix()-1, 0)
	}

	return timeNow
}
