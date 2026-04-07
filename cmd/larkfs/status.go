package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/IchenDEV/larkfs/pkg/daemon"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show mount point status",
		RunE: func(cmd *cobra.Command, args []string) error {
			mounts, err := daemon.ListMounts()
			if err != nil {
				return fmt.Errorf("failed to list mounts: %w", err)
			}

			if len(mounts) == 0 {
				fmt.Println("No active mounts.")
				return nil
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(mounts)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "MOUNTPOINT\tPID\tUPTIME\tBACKEND\tSTATUS")
			for _, m := range mounts {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
					m.Mountpoint, m.PID, m.Uptime(), m.Backend, m.Status())
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}
