/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package testutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const defaultTimeFormat = "-20062006-01-02-1504-05"

// MkTestDir creates a test directory in os.TempDir()/<name>-<timePrefix>
// It returns the path to the directory created.
func MkTestDir(name string) (string, error) {
	dirName := filepath.Join(os.TempDir(), time.Now().Format("logfellertest-"+name+defaultTimeFormat))
	return dirName, os.Mkdir(dirName, 0700)
}

// TrueOrFatal fails t if cond is false and terminates the run of the test
func TrueOrFatal(t testing.TB, cond bool, fmtmsg string, args ...interface{}) {
	if !cond {
		t.Fatalf(fmtmsg, args...)
	}
}

// TrueOrError fails t if cond is false but will not exit the test. Returns !cond
func TrueOrError(t testing.TB, cond bool, fmtmsg string, args ...interface{}) bool {
	if !cond {
		t.Errorf(fmtmsg, args...)
	}
	return !cond
}

// Time related utils

func DateOfYear(t time.Time, month time.Month, day, hour, min, sec int) time.Time {
	return time.Date(t.Year(), month, day, hour, min, sec, 0, t.Location())
}

func TimeOfDay(t time.Time, hour, min, sec int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, min, sec, 0, t.Location())
}
