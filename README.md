# logfeller [![Go Reference](https://pkg.go.dev/badge/github.com/lohvht/logfeller.svg)](https://pkg.go.dev/github.com/lohvht/logfeller) [![Build Status](https://travis-ci.com/lohvht/logfeller.svg?branch=main)](https://travis-ci.com/lohvht/logfeller) [![codecov](https://codecov.io/gh/lohvht/logfeller/branch/main/graph/badge.svg?token=Y988UGIV2D)](https://codecov.io/gh/lohvht/logfeller)

## Installation
```
go get github.com/lohvht/logfeller
```

## Overview

Logfeller provides a file handler that rotates files on a rolling schedule. Logfeller is inspired by [lumberjack](https://github.com/natefinch/lumberjack) but serves a different niche. It handles which file to write to based on a rotational schedule instead of the file's max size.


## Getting Started

`*logfeller.File` implements the [io.Writer](https://golang.org/pkg/io/#Writer) interface. This opens it up to the possibility of it being used in places where an `io.Writer` is needed, such as the the standard library's [log](https://golang.org/pkg/log)

### Using `*logfeller.File`

Logfeller contains `json` and `yaml` bindings so you may use [json.Unmarshal](https://golang.org/pkg/encoding/json/#Unmarshal) and [yaml.Unmarshal](https://pkg.go.dev/gopkg.in/yaml.v2) to marshal it into `*logfeller.File`

```
// File is the rotational file handler. It writes to the filename specified
// and will rotate based on the schedule passed in.
type File struct {
	// Filename is the filename to write to. If empty, uses the filename
	// `<cmdname>-logfeller.log` within os.TempDir()
	Filename string `json:"filename" yaml:"filename"`
	// When tells the logger to rotate the file, it is case insensitive.
	// Currently supported values are
	// 	"h" - hour
	// 	"d" - day
	// 	"m" - month
	// 	"y" - year
	When WhenRotate `json:"when" yaml:"when"`
	// RotationSchedule defines the when the rotation should be occur.
	// The values that should be passed into depends on the When field.
	// If When is:
	// 	"h" - pass in strings of format "04:05" (MM:SS)
	// 	"d" - pass in strings of format "1504:05" (HHMM:SS)
	// 	"m" - pass in strings of format "02 1504:05" (DD HHMM:SS)
	// 	"y" - pass in strings of format "0102 1504:05" (mmDD HHMM:SS)
	// where mm, DD, HH, MM, SS represents month, day, hour, minute
	// and seconds respectively.
	// If RotationSchedule is empty, a sensible default is depending on `When`
	// will be used instead.
	// If When is:
	// 	"h" - "00:00" will be used (rotate on the 0th minute, 0th second of the hour)
	// 	"d" - "0000:00" will be used (rotate at 12am daily)
	// 	"m" - "01 0000:00" will be used (rotate on the 1st day at 12am monthly)
	// 	"y" - "0101 0000:00" will be used (rotate on 1st Jan at 12am every year)
	RotationSchedule []string `json:"rotation_schedule" yaml:"rotation-schedule"`
	// UseLocal determines if the time used to rotate is based on the system's
	// local time
	UseLocal bool `json:"use_local" yaml:"use-local"`
	// Backups maintains the number of backups to keep. If this is empty, do
	// not delete backups.
	Backups int `json:"backups" yaml:"backups"`
	// BackupTimeFormat is time format used for the backup file's encoded timestamp.
	// Defaults to ".2006-01-02T1504-05" if empty.
	// See the golang `time` package for more example formats
	// https://golang.org/pkg/time/#Time.Format
	BackupTimeFormat string `json:"backup_time_format" yaml:"backup-time-format"`
	// contains filtered or unexported fields
}
```

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

### Rotational Logic
Logfeller's rotational logic depends on the `When` value and `RotationalSchedule` specified.

For example, if `When` is `"d"` and RotationSchedule is `[]string{"0000:00", "1430:00"}`, This means that we would like to rotate daily at midnight and 2:30pm.

Using the same example above, upon writing to the file after midnight, Logfeller will check if it should rotate. If it has not rotated recently, logfeller will attempt to backup the file. The backup process renames the current file to one that contains a timestamp of the previous rotation time. If the backup file exists, the contents of the current file is written to the backup and then the current file will be deleted. After the backup, a new file will be created using the original file name.

The format of the timestamp on the backup file will be based on BackupTimeFormat specified.

### Backup Files

Backups use the log file name given in the form `<name><timestamp><ext>` where name is the filename given without extension, timestamp is previous rotate time formatted with the BackupTimeFormat given and extension is the original extension.

Whenever a new file is created, older backups may be cleared. The most recent files based on the timestamp encoded with BackupTimeFormat will be retained up to the number of Backups specified. If Backups is 0, no old backups will be deleted.
