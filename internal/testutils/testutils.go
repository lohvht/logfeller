/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package testutils

import (
	"os"
	"path/filepath"
	"time"
)

const defaultTimeFormat = "20062006-01-02-1504"

// MkTestDir creates a test directory in os.TempDir()/<name>-<timePrefix>
// It returns the path to the directory created.
func MkTestDir(name string) (string, error) {
	dirName := filepath.Join(os.TempDir(), time.Now().Format(name+defaultTimeFormat))
	return dirName, os.Mkdir(dirName, 0700)
}
