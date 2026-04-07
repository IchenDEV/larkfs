package main

import (
	"fmt"
	"log/slog"

	"github.com/IchenDEV/larkfs/pkg/daemon"
	"github.com/spf13/cobra"
)

func newUnmountCmd() *cobra.Command {
	var (
		all     bool
		force   bool
		timeout int
	)

	cmd := &cobra.Command{
		Use:     "unmount [mountpoint]",
		Aliases: []string{"umount"},
		Short:   "Unmount a Lark filesystem",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return daemon.UnmountAll(force, timeout)
			}
			if len(args) == 0 {
				return fmt.Errorf("mountpoint required (or use --all)")
			}
			return daemon.Unmount(args[0], force, timeout)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&all, "all", false, "Unmount all mount points")
	f.BoolVar(&force, "force", false, "Force unmount (SIGKILL + fusermount -uz)")
	f.IntVar(&timeout, "timeout", 10, "Seconds to wait for in-flight operations")

	return cmd
}

func init() {
	_ = slog.Default()
}
