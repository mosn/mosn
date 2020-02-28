// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

import (
	"gopkg.in/yaml.v2"
)

type confProperty struct {
	Name  string `xml:"name,attr" json:"name,omitempty" yaml:"name"`
	Value string `xml:",chardata" json:"value,omitempty"  yaml:"value"`
}

type kvFilter struct {
	Enabled  string         `xml:"enabled,attr" json:"enabled,omitempty" yaml:"enabled"`
	Tag      string         `xml:"tag" json:"tag,omitempty" yaml:"tag"`
	Level    string         `xml:"level" json:"level,omitempty" yaml:"level"`
	Type     string         `xml:"type" json:"type,omitempty"  yaml:"type"`
	Property []confProperty `xml:"property" json:"properties,omitempty" yaml:"properties"`
}

type loggerConfig struct {
	Filter []kvFilter `xml:"filter" json:"filters,omitempty" yaml:"filters"`
}

type logConfig struct {
	Logging loggerConfig `json:"logging,omitempty" yaml:"logging"`
}

func loadConfFile(filename string) *loggerConfig {
	// Open the configuration file
	fd, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"LoadConfiguration: Error: Could not open %q for reading: %s\n",
			filename, err)
		os.Exit(1)
	}
	defer fd.Close()

	contents, err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"LoadConfiguration: Error: Could not read %q: %s\n",
			filename, err)
		os.Exit(1)
	}

	lc := new(loggerConfig)
	if strings.HasSuffix(filename, ".yml") {
		yc := new(logConfig)
		if err := yaml.Unmarshal(contents, yc); err != nil {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Could not parse YAML configuration in %q: %s\n",
				filename, err)
			os.Exit(1)
		}
		lc = &yc.Logging
	} else if strings.HasSuffix(filename, ".json") {
		jc := new(logConfig)
		if err := json.Unmarshal(contents, jc); err != nil {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Could not parse JSON configuration in %q: %s\n",
				filename, err)
			os.Exit(1)
		}
		lc = &jc.Logging

	} else {
		if err := xml.Unmarshal(contents, lc); err != nil {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Could not parse XML configuration in %q: %s\n",
				filename, err)
			os.Exit(1)
		}
	}

	return lc
}

// Load XML configuration; see examples/example.xml or examples.json for documentation
func (log *Logger) LoadConfiguration(filename string) Logger {
	log.Close()
	log.minLevel = CRITICAL

	lc := loadConfFile(filename)
	for _, filter := range lc.Filter {
		var filt LogWriter
		var lvl Level
		bad, good, enabled := false, true, false

		// Check required children
		if len(filter.Enabled) == 0 {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Required attribute %s for filter missing in %s\n",
				"enabled", filename)
			bad = true
		} else {
			enabled = filter.Enabled != "false"
		}
		if len(filter.Tag) == 0 {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Required child <%s> for filter missing in %s\n",
				"tag", filename)
			bad = true
		}
		if len(filter.Type) == 0 {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Required child <%s> for filter missing in %s\n",
				"type", filename)
			bad = true
		} else {
			if filter.Type == "record" {
				continue
			}
		}
		if len(filter.Level) == 0 {
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Required child <%s> for filter missing in %s\n",
				"level", filename)
			bad = true
		}

		filterLevel := strings.ToLower(filter.Level)
		switch filterLevel {
		case "finest":
			lvl = FINEST
		case "fine":
			lvl = FINE
		case "debug":
			lvl = DEBUG
		case "trace":
			lvl = TRACE
		case "info":
			lvl = INFO
		case "warning":
			lvl = WARNING
		case "error":
			lvl = ERROR
		case "critical":
			lvl = CRITICAL
		default:
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Required child <%s> for filter has unknown value in %s: %s\n",
				"level", filename, filter.Level)
			bad = true
		}

		// Just so all of the required attributes are errored at the same time if missing
		if bad {
			os.Exit(1)
		}
		if lvl < log.minLevel {
			log.minLevel = lvl
		}

		switch filter.Type {
		case "console":
			filt, good = confToConsoleLogWriter(filename, filter.Property, enabled)
		case "file":
			filt, good = confToFileLogWriter(filename, filter.Property, enabled)
		case "xml":
			filt, good = confToXMLLogWriter(filename, filter.Property, enabled)
		case "socket":
			filt, good = confToSocketLogWriter(filename, filter.Property, enabled)
		default:
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Error: Could not load XML configuration in %s: unknown filter type \"%s\"\n",
				filename, filter.Type)
			os.Exit(1)
		}

		// Just so all of the required params are errored at the same time if wrong
		if !good {
			os.Exit(1)
		}

		// If we're disabled (syntax and correctness checks only), don't add to logger
		if !enabled {
			continue
		}

		//log.FilterMap[filter.Tag] = &Filter{lvl, filt}
		log.AddFilter(filter.Tag, lvl, filt)
	}

	return *log
}

func confToConsoleLogWriter(filename string, props []confProperty, enabled bool) (*ConsoleLogWriter, bool) {
	jsonFormat := false
	format := "[%D %T] [%L] (%S) %M"
	callerFlag := true

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "json":
			jsonFormat = strings.Trim(prop.Value, " \r\n") != "false"
		case "format":
			format = strings.Trim(prop.Value, " \r\n")
		case "caller":
			// 为了兼容以往设置，默认 caller 为 true，只有明确设置其为 "false" 时，才设置其为 false
			callerFlag = !(strings.Trim(prop.Value, " \r\n") == "false")
		default:
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Warning: Unknown property \"%s\" for console filter in %s\n",
				prop.Name, filename)
		}
	}

	// If it's disabled, we're just checking syntax
	if !enabled {
		return nil, true
	}

	clw := NewConsoleLogWriter(jsonFormat)
	clw.SetFormat(format)
	clw.SetCallerFlag(callerFlag)

	return clw, true
}

// Parse a number with K/M/G suffixes based on thousands (1000) or 2^10 (1024)
func strToNumSuffix(str string, mult int) int {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, _ := strconv.Atoi(str)
	return parsed * num
}

func strToInt(str string) int {
	n, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return n
}
func strToI64(str string) int64 {
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func confToFileLogWriter(filename string, props []confProperty, enabled bool) (*FileLogWriter, bool) {
	file := ""
	jsonformat := false
	format := "[%D %T] [%L] (%S) %M"
	maxlines := 0
	maxsize := int64(0)
	maxbackup := 0
	rotate := false
	daily := false
	rothours := 0
	bufSize := 0
	callerFlag := true

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
		case "json":
			jsonformat = strings.Trim(prop.Value, " \r\n") != "false"
		case "format":
			format = strings.Trim(prop.Value, " \r\n")
		case "maxlines":
			maxlines = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = int64(strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024))
		case "bufsize":
			bufSize = strToInt(strings.Trim(prop.Value, " \r\n"))
		case "maxbackup":
			maxbackup = strToInt(strings.Trim(prop.Value, " \r\n"))
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "hourly":
			rothours, _ = strconv.Atoi(strings.Trim(prop.Value, " \r\n"))
		case "caller":
			// 为了兼容以往设置，默认 caller 为 true，只有明确设置其为 "false" 时，才设置其为 false
			callerFlag = !(strings.Trim(prop.Value, " \r\n") == "false")

		default:
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n",
				prop.Name, filename)
		}
	}

	// Check properties
	if len(file) == 0 {
		fmt.Fprintf(os.Stderr,
			"LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n",
			"filename", filename)
		return nil, false
	}

	// If it's disabled, we're just checking syntax
	if !enabled {
		return nil, true
	}

	flw := NewFileLogWriter(file, rotate, bufSize)
	flw.SetJson(jsonformat)
	flw.SetFormat(format)
	flw.SetRotateLines(maxlines)
	flw.SetRotateSize(maxsize)
	flw.SetRotateMaxBackup(maxbackup)
	flw.SetRotateDaily(daily)
	flw.SetRotateHourly(rothours)
	flw.SetCallerFlag(callerFlag)
	return flw, true
}

func confToXMLLogWriter(filename string, props []confProperty, enabled bool) (*FileLogWriter, bool) {
	file := ""
	maxrecords := 0
	maxsize := int64(0)
	maxbackup := 0
	daily := false
	rothours := 0
	rotate := false
	bufSize := 0
	callerFlag := true

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
		case "maxrecords":
			maxrecords = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = int64(strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024))
		case "bufsize":
			bufSize = strToInt(strings.Trim(prop.Value, " \r\n"))
		case "maxbackup":
			maxbackup = strToInt(strings.Trim(prop.Value, " \r\n"))
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "hourly":
			rothours, _ = strconv.Atoi(strings.Trim(prop.Value, " \r\n"))
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		case "caller":
			// 为了兼容以往设置，默认 caller 为 true，只有明确设置其为 "false" 时，才设置其为 false
			callerFlag = !(strings.Trim(prop.Value, " \r\n") == "false")
		default:
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Warning: Unknown property \"%s\" for filter in %s\n",
				prop.Name, filename)
		}
	}

	// Check properties
	if len(file) == 0 {
		fmt.Fprintf(os.Stderr,
			"LoadConfiguration: Error: Required property \"%s\" for filter missing in %s\n",
			"filename", filename)
		return nil, false
	}

	// If it's disabled, we're just checking syntax
	if !enabled {
		return nil, true
	}

	xlw := NewXMLLogWriter(file, rotate, bufSize)
	xlw.SetRotateLines(maxrecords)
	xlw.SetRotateSize(maxsize)
	xlw.SetRotateMaxBackup(maxbackup)
	xlw.SetRotateDaily(daily)
	xlw.SetRotateHourly(rothours)
	xlw.SetCallerFlag(callerFlag)

	return xlw, true
}

func confToSocketLogWriter(filename string, props []confProperty, enabled bool) (*SocketLogWriter, bool) {
	endpoint := ""
	protocol := "udp"
	callerFlag := true

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "endpoint":
			endpoint = strings.Trim(prop.Value, " \r\n")
		case "protocol":
			protocol = strings.Trim(prop.Value, " \r\n")
		case "caller":
			// 为了兼容以往设置，默认 caller 为 true，只有明确设置其为 "false" 时，才设置其为 false
			callerFlag = !(strings.Trim(prop.Value, " \r\n") == "false")
		default:
			fmt.Fprintf(os.Stderr,
				"LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n",
				prop.Name, filename)
		}
	}

	// Check properties
	if len(endpoint) == 0 {
		fmt.Fprintf(os.Stderr,
			"LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n",
			"endpoint", filename)
		return nil, false
	}

	// If it's disabled, we're just checking syntax
	if !enabled {
		return nil, true
	}

	writer := NewSocketLogWriter(protocol, endpoint)
	writer.SetCallerFlag(callerFlag)

	return writer, true
}
