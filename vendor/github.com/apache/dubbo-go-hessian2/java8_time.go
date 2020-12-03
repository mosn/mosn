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

package hessian

import (
	"github.com/apache/dubbo-go-hessian2/java8_time"
)

func init() {
	RegisterPOJO(&java8_time.Year{Year: 2020})
	RegisterPOJO(&java8_time.YearMonth{Month: 2020, Year: 6})
	RegisterPOJO(&java8_time.Period{Years: 2020, Months: 6, Days: 16})
	RegisterPOJO(&java8_time.LocalDate{Year: 2020, Month: 6, Day: 16})
	RegisterPOJO(&java8_time.LocalTime{Hour: 6, Minute: 5, Second: 4, Nano: 3})
	RegisterPOJO(&java8_time.LocalDateTime{Date: java8_time.LocalDate{Year: 2020, Month: 6, Day: 16}, Time: java8_time.LocalTime{Hour: 6, Minute: 5, Second: 4, Nano: 3}})
	RegisterPOJO(&java8_time.MonthDay{Month: 6, Day: 16})
	RegisterPOJO(&java8_time.Duration{Seconds: 30, Nanos: 10})
	RegisterPOJO(&java8_time.Instant{Seconds: 100, Nanos: 10})
	RegisterPOJO(&java8_time.ZoneOffSet{Seconds: 7200})
	RegisterPOJO(&java8_time.OffsetDateTime{DateTime: java8_time.LocalDateTime{Date: java8_time.LocalDate{Year: 2020, Month: 6, Day: 16}, Time: java8_time.LocalTime{Hour: 6, Minute: 5, Second: 4, Nano: 3}}, Offset: java8_time.ZoneOffSet{Seconds: 7200}})
	RegisterPOJO(&java8_time.OffsetTime{LocalTime: java8_time.LocalTime{Hour: 6, Minute: 5, Second: 4, Nano: 3}, ZoneOffset: java8_time.ZoneOffSet{Seconds: 7200}})
	RegisterPOJO(&java8_time.ZonedDateTime{DateTime: java8_time.LocalDateTime{Date: java8_time.LocalDate{Year: 2020, Month: 6, Day: 16}, Time: java8_time.LocalTime{Hour: 6, Minute: 5, Second: 4, Nano: 3}}, Offset: java8_time.ZoneOffSet{Seconds: 7200}, ZoneId: "Z"})

}
