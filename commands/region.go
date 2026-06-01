package commands

import (
	"github.com/abiosoft/ishell"
	"github.com/bwhaley/ssmsh/parameterstore"
)

const regionUsage string = `
usage: region region
Update your region.
Example:
region us-west-2
`

func region(c *ishell.Context) {
	if len(c.Args) == 0 {
		if ps.Region != "" {
			shell.Println(ps.Region)
		}
	} else if len(c.Args) == 1 {
		cwd := ps.Cwd
		ps.Region = c.Args[0]
		err := ps.NewParameterStore(true)
		if err != nil {
			shell.Printf("Error: %s", err)
		} else {
			// Preserve the working directory across the region change when the
			// path also exists in the new region; otherwise fall back to root.
			if err := ps.SetCwd(parameterstore.ParameterPath{Name: cwd, Region: ps.Region}); err != nil {
				ps.Cwd = parameterstore.Delimiter
			}
			setPrompt(ps.Cwd)
		}
	}
}
