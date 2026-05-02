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
		Use:           "larkfs",
		Short:         "Virtual filesystem for Lark/Feishu",
		Long:          "Mount Lark/Feishu Drive, Wiki, IM, Calendar, Tasks, Mail, Meetings as a local filesystem via FUSE or WebDAV.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(
		newMountCmd(),
		newUnmountCmd(),
		newServeCmd(),
		newStatusCmd(),
		newDoctorCmd(),
		newInitCmd(),
		newVersionCmd(),
		newNativeCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
