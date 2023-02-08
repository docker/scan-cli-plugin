package internal

import (
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/docker/cli/cli/command"
	"github.com/fatih/color"
	"golang.org/x/term"
)

const (
	message = "> The `docker scan` **command is deprecated** and will no longer be supported after April 13, 2023.  \n" +
		"> Run the `docker scout cves` command to continue to get vulnerabilities on your images or install the Snyk CLI.  \n" +
		"> See https://www.docker.com/products/docker-scout for more details."
)

func PrintDeprecationMessage(cli command.Cli) {
	r := getTermRenderer()
	str, err := r.Render(message)
	if err != nil {
		_, _ = fmt.Fprintln(cli.Err(), message)
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
