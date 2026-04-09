package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:   "larkfs",
		Short: "Virtual filesystem for Lark/Feishu",
		Long:  "Mount Lark/Feishu Drive, Wiki, IM, Calendar, Tasks, Mail, Meetings as a local filesystem via FUSE or WebDAV.",
	}

	root.AddCommand(
		newMountCmd(),
		newUnmountCmd(),
		newServeCmd(),
		newStatusCmd(),
		newDoctorCmd(),
		newInitCmd(),
		newVersionCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("larkfs %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}
