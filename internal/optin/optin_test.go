// +build !windows

/*
   Copyright 2020 Docker Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package optin

import (
	"bytes"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
)

// TODO: this test is skipped (not built) on windows platform, as github.com/Netflix/go-expect
// relies on github.com/creack/pty which doesn't provide any windows support. We should find
// another way to test this feature on all platforms.
func TestAskForConsent(t *testing.T) {
	buf := new(bytes.Buffer)
	console, _, err := vt10x.NewVT10XConsole(expect.WithStdout(buf))
	require.Nil(t, err)
	defer console.Close() //nolint:errcheck

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		_, err := console.Expect(expect.WithTimeout(100*time.Millisecond),
			expect.String("Docker Scan relies upon access to Snyk a third party provider, do you consent to proceed using Snyk?"))
		assert.NilError(t, err)
		_, err = console.SendLine("y")
		assert.NilError(t, err)
		_, err = console.ExpectEOF()
		assert.NilError(t, err)
	}()

	answer, err := AskForConsent(terminal.Stdio{
		In:  console.Tty(),
		Out: console.Tty(),
		Err: console.Tty(),
	})
	assert.NilError(t, err)
	assert.Equal(t, answer, true)

	assert.NilError(t, console.Tty().Close())
	<-donec
}
