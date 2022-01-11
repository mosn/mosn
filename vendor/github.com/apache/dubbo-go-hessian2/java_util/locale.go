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

package java_util

// LocaleEnum is Locale enumeration value
type LocaleEnum int

// Locale struct enum
const (
	ENGLISH LocaleEnum = iota
	FRENCH
	GERMAN
	ITALIAN
	JAPANESE
	KOREAN
	CHINESE
	SIMPLIFIED_CHINESE
	TRADITIONAL_CHINESE
	FRANCE
	GERMANY
	ITALY
	JAPAN
	KOREA
	CHINA
	PRC
	TAIWAN
	UK
	US
	CANADA
	CANADA_FRENCH
	ROOT
)

// Locale => java.util.Locale
type Locale struct {
	// ID is used to implement enumeration
	id     LocaleEnum
	lang   string
	county string
}

func (locale *Locale) County() string {
	return locale.county
}

func (locale *Locale) Lang() string {
	return locale.lang
}

func (locale *Locale) String() string {
	if len(locale.county) != 0 {
		return locale.lang + "_" + locale.county
	}
	return locale.lang
}

// LocaleHandle => com.alibaba.com.caucho.hessian.io.LocaleHandle object
type LocaleHandle struct {
	Value string `hessian:"value"`
}

func (LocaleHandle) JavaClassName() string {
	return "com.alibaba.com.caucho.hessian.io.LocaleHandle"
}

// locales is all const Locale struct slice
// localeMap is key = locale.String() value = locale struct
var (
	locales   []Locale            = make([]Locale, 22, 22)
	localeMap map[string](Locale) = make(map[string](Locale), 22)
)

// init java.util.Locale static object
func init() {
	locales[ENGLISH] = Locale{
		id:     ENGLISH,
		lang:   "en",
		county: "",
	}
	locales[FRENCH] = Locale{
		id:     FRENCH,
		lang:   "fr",
		county: "",
	}
	locales[GERMAN] = Locale{
		id:     GERMAN,
		lang:   "de",
		county: "",
	}
	locales[ITALIAN] = Locale{
		id:     ITALIAN,
		lang:   "it",
		county: "",
	}
	locales[JAPANESE] = Locale{
		id:     JAPANESE,
		lang:   "ja",
		county: "",
	}
	locales[KOREAN] = Locale{
		id:     KOREAN,
		lang:   "ko",
		county: "",
	}
	locales[CHINESE] = Locale{
		id:     CHINESE,
		lang:   "zh",
		county: "",
	}
	locales[SIMPLIFIED_CHINESE] = Locale{
		id:     SIMPLIFIED_CHINESE,
		lang:   "zh",
		county: "CN",
	}
	locales[TRADITIONAL_CHINESE] = Locale{
		id:     TRADITIONAL_CHINESE,
		lang:   "zh",
		county: "TW",
	}
	locales[FRANCE] = Locale{
		id:     FRANCE,
		lang:   "fr",
		county: "FR",
	}
	locales[GERMANY] = Locale{
		id:     GERMANY,
		lang:   "de",
		county: "DE",
	}
	locales[ITALY] = Locale{
		id:     ITALY,
		lang:   "it",
		county: "it",
	}
	locales[JAPAN] = Locale{
		id:     JAPAN,
		lang:   "ja",
		county: "JP",
	}
	locales[KOREA] = Locale{
		id:     KOREA,
		lang:   "ko",
		county: "KR",
	}
	locales[CHINA] = locales[SIMPLIFIED_CHINESE]
	locales[PRC] = locales[SIMPLIFIED_CHINESE]
	locales[TAIWAN] = locales[TRADITIONAL_CHINESE]
	locales[UK] = Locale{
		id:     UK,
		lang:   "en",
		county: "GB",
	}
	locales[US] = Locale{
		id:     US,
		lang:   "en",
		county: "US",
	}
	locales[CANADA] = Locale{
		id:     CANADA,
		lang:   "en",
		county: "CA",
	}
	locales[CANADA_FRENCH] = Locale{
		id:     CANADA_FRENCH,
		lang:   "fr",
		county: "CA",
	}
	locales[ROOT] = Locale{
		id:     ROOT,
		lang:   "",
		county: "",
	}
	for _, locale := range locales {
		localeMap[locale.String()] = locale
	}
}

// ToLocale get locale from enum
func ToLocale(e LocaleEnum) *Locale {
	return &locales[e]
}

// GetLocaleFromHandler is use LocaleHandle get Locale
func GetLocaleFromHandler(localeHandler *LocaleHandle) *Locale {
	locale := localeMap[localeHandler.Value]
	return &locale
}
