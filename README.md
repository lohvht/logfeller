# logfeller [![Go Reference](https://pkg.go.dev/badge/github.com/lohvht/logfeller.svg)](https://pkg.go.dev/github.com/lohvht/logfeller) 

## Installation
```
go get github.com/lohvht/logfeller
```

## Overview

Logfeller provides a file handler that rotates files on a rolling schedule. Logfeller is inspired by [lumberjack](https://github.com/natefinch/lumberjack) but serves a different niche. Logfeller handles which file to write to based on a rotational schedule instead of using a max size (such as every day at noon etc.)

Logfeller is meant to be a small pluggable component in a logging stack. It is intended to be a pluggable component that controls how files are written and rotated. to be a small pluggable component in a logging stack. It is intended to be a pluggable component that controls how files are written and rotated.

## Getting Started

`*logfeller.File` implements the [io.Writer](https://golang.org/pkg/io/#Writer). This opens it up to the possibility of it being used in places where an `io.Writer` is needed, such as the the standard library's [log](https://golang.org/pkg/log)

### Using `*logfeller.File`

```
type File struct {
	// Filename is the filename to write to. Uses the filename
	// `<cmdname>-logfeller.log` in os.TempDir() if empty.
	Filename string `json:"filename" yaml:"filename"`
	// When tells the logger to rotate the file, it is case insensitive.
	// Currently supported values are
	// 	"h" - hour
	// 	"d" - day
	// 	"m" - month
	// 	"y" - year
	When WhenRotate `json:"when" yaml:"when"`
	// RotationSchedule defines the exact time that the rotator should be
	// rotating. The values that should be passed into depends on the When field.
	// If When is:
	// 	"h" - pass in strings of format "04:05" (MM:SS)
	// 	"d" - pass in strings of format "1504:05" (HHMM:SS)
	// 	"m" - pass in strings of format "02 1504:05" (DD HHMM:SS)
	// 	"y" - pass in strings of format "0102 1504:05" (mmDD HHMM:SS)
	// where mm, DD, HH, MM, SS represents month, day, hour, minute
	// and seconds respectively.
	// If RotationSchedule is empty, a sensible default will be used instead.
	RotationSchedule []string `json:"rotation_schedule" yaml:"rotation-schedule"`
	// UseLocal determines if the time used to rotate is based on the system's
	// local time
	UseLocal bool `json:"use_local" yaml:"use-local"`
	// Backups maintains the number of backups to keep. If this is empty, do
	// not delete backups.
	Backups int `json:"backups" yaml:"backups"`
	// BackupTimeFormat is the backup time format used when logfeller rotates
	// the file. Defaults to ".2006-01-02T1504-05" if empty
	// See the golang `time` package for more example formats
	// https://golang.org/pkg/time/#Time.Format
	BackupTimeFormat string `json:"backup_time_format" yaml:"backup-time-format"`
	// contains filtered or unexported fields
}
```

Logfeller contains `json` and `yaml` bindings so you may use [json.Unmarshal](https://golang.org/pkg/encoding/json/#Unmarshal) and [yaml.Unmarshal](https://pkg.go.dev/gopkg.in/yaml.v2) to marshal it into `*logfeller.File`

An example of how to unmarshal JSON into logfeller is shown below:
```
package main

import (
	"encoding/json"
	"log"

	"github.com/lohvht/logfeller"
)

func main() {
	var f logfeller.File
	err := json.Unmarshal([]byte(`{
		"filename":         "some-file-name.txt",
		"when":             "D",
		"rotation_schedule": ["0000:00", "1430:00"],
		"use_local":         true,
		"backups":          30,
		"backup_time_format": "Jan _2 15:04:05"
}`), &f)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.SetOutput(&f)
	// more code ...
}
```

## What does *logfeller.File do?

*logfeller.File is an io.Writer that writes to the specified filename.

*logfeller.File opens or creates the file on the first Write. If such a file exists, logfeller will check the file's modtime and rotate if it is due for rotation, otherwise, it will open and append to that file. 

### Rotational Logic
Logfeller's rotational logic is Based on the `When` value specified. There are 4 values currently supported, `"h"`, `"d"`, `"m"` and `"y"`. This tells Logfeller to rotate hourly, daily, monthly or yearly respectively.

Based on the When value specified, logfeller will then check the RotationalSchedule. For example, If we choose to rotate daily and the rotational schedule provided is `"0000:00"` and `"1430:00"`, this tells logfeller to rotate at midnight and 2:30 pm. Using the same example, When we try to write to the file after midnight and we have yet to rotate yet, logfeller will rename the file by putting a timestamp of the previous rotate time as part of the filename, creating a backup. If a backup file of the previous rotate time already exists, logfeller will append the current file's content to that previous log file instead and the current file will be removed. After that, a new log file will be recreated using the original file name.

The format of the backup file will be based on BackupTimeFormat specified.

## Clearing Old Log Files

Logfeller may clear older backups if Backups > 0. The most recent files based on the timestamp encoded with BackupTimeFormat will be retained up to the number of Backups specified, the rest of those files will be removed. If Backups is 0, logfeller will not remove any files at all.
