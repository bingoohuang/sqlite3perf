package sqlite3perf

import (
	"fmt"
	"github.com/bingoohuang/gg/pkg/v"
	"github.com/spf13/cobra"
)

// VersionCmd is the struct representing version sub-command.
type VersionCmd struct{}

func init() {
	c := VersionCmd{}
	cmd := &cobra.Command{
		Use:   "version",
		Short: "print version information",
		Run:   c.run,
	}

	rootCmd.AddCommand(cmd)
}

func (g *VersionCmd) run(*cobra.Command, []string) {
	fmt.Println(v.Version())
}
