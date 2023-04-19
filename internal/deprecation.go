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

package internal

import (
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/docker/cli/cli/command"
	"github.com/fatih/color"
	"golang.org/x/term"
)

var (
	eolMessage = fmt.Sprintf(`
The %s command has been removed.

To continue learning about the vulnerabilities of your images, and many other features, use the new %s command.
Run %s, or learn more at %s
`, "`docker scan`", "`docker scout`", "`docker scout --help`", "https://docs.docker.com/engine/reference/commandline/scout/")
)

func PrintEOLMessage(cli command.Cli) {
	r := getTermRenderer()
	str, err := r.Render(eolMessage)
	if err != nil {
		_, _ = fmt.Fprintln(cli.Err(), eolMessage)
	} else {
		_, _ = fmt.Fprintln(cli.Err(), str)
	}
}

func getTermRenderer() *glamour.TermRenderer {
	w, _, err := term.GetSize(0)
	if err != nil {
		w = 80
	}

	var r *glamour.TermRenderer
	if color.NoColor {
		r, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("notty"),
			glamour.WithWordWrap(w-10))
	} else {
		r, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(w-10))
	}
	return r
}
