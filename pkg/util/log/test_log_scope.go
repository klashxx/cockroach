// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Raphael 'kena' Poss (knz@cockroachlabs.com)

package log

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/util/caller"
)

// TestLogScope represents the lifetime of a logging output.  It
// ensures that the log files are stored in a directory specific to a
// test, and asserts that logging output is not written to this
// directory beyond the lifetime of the scope.
type TestLogScope string

// tShim is the part of testing.T used by TestLogScope.
// We can't use testing.T directly because we have
// a linter which forbids its use in public interfaces.
type tShim interface {
	Fatal(...interface{})
	Failed() bool
	Error(...interface{})
	Errorf(fmt string, args ...interface{})
}

// Scope creates a TestLogScope which corresponds to the lifetime of a
// logging directory. If testName is empty, the logging directory is
// named after the caller of Scope, up `skip` caller levels. It also
// disables logging to stderr for severity levels below ERROR.
func Scope(t tShim, testName string) TestLogScope {
	if testName == "" {
		testName = "logUnknown"
		if _, _, f := caller.Lookup(1); f != "" {
			parts := strings.Split(f, ".")
			testName = "log" + parts[len(parts)-1]
		}
	}
	tempDir, err := ioutil.TempDir("", testName)
	if err != nil {
		t.Fatal(err)
	}
	if err := dirTestOverride(tempDir); err != nil {
		t.Fatal(err)
	}
	if err := EnableLogFileOutput(tempDir, Severity_ERROR); err != nil {
		t.Fatal(err)
	}
	return TestLogScope(tempDir)
}

// Close cleans up a TestLogScope. The directory and its contents are
// deleted, unless the test has failed and the directory is non-empty.
func (l TestLogScope) Close(t tShim) {
	if string(l) == "" {
		// Never initialized.
		return
	}
	defer func() {
		// Check whether there is something to remove.
		emptyDir, err := isDirEmpty(string(l))
		if err != nil {
			t.Fatal(err)
		}
		if t.Failed() && !emptyDir {
			// If the test failed, we keep the log files for further investigation,
			// but only if there were any.
			t.Errorf("test log files left over in: %s", l)
		} else {
			// Clean up.
			if err := os.RemoveAll(string(l)); err != nil {
				t.Error(err)
			}
		}
	}()
	// Flush/Close the log files.
	if err := dirTestOverride(""); err != nil {
		t.Fatal(err)
	}
}

// dirTestOverride sets the default value for the logging output directory
// for use in tests.
func dirTestOverride(dir string) error {
	// Ensure any remaining logs are written.
	Flush()

	logDir.Lock()
	logDir.name = dir
	logDir.Unlock()

	// When we change the directory we close the current logging
	// output, so that a rotation to the new directory is forced on
	// the next logging event.
	logging.mu.Lock()
	err := logging.closeFilesLocked()
	logging.mu.Unlock()

	return err
}

func isDirEmpty(dirname string) (bool, error) {
	f, err := os.Open(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	list, err := f.Readdir(1)
	errClose := f.Close()
	if err != nil && err != io.EOF {
		return false, err
	}
	if errClose != nil {
		return false, errClose
	}
	return len(list) == 0, nil
}
