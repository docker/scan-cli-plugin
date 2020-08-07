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
	"fmt"
	"io"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

// AskForConsent prompts a consent question to inform about Snyk usage on behalf
func AskForConsent(stdio terminal.Stdio) (bool, error) {
	answer := false
	prompt := &survey.Confirm{
		Message: "Docker Scan relies upon access to Snyk, a third party provider, do you consent to proceed using Snyk?",
	}
	if err := survey.AskOne(prompt, &answer, survey.WithStdio(stdio.In, stdio.Out, stdio.Err)); err != nil {
		return false, fmt.Errorf("failed to ask user consent: %s", err)
	}
	return answer, nil
}

// TerminalStdio generate a terminal.Stdio from the command input/output
func TerminalStdio(in io.Reader, out, err io.Writer) terminal.Stdio {
	return terminal.Stdio{
		In:  termInput{in},
		Out: termOutput{out},
		Err: err,
	}
}

type termInput struct {
	io.Reader
}

func (termInput) Fd() uintptr {
	return 1
}

type termOutput struct {
	io.Writer
}

func (termOutput) Fd() uintptr {
	return 1
}
