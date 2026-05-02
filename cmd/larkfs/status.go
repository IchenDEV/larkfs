package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/IchenDEV/larkfs/pkg/daemon"
	"github.com/spf13/cobra"
)

type statusMount struct {
	PID        int      `json:"pid"`
	Mountpoint string   `json:"mountpoint"`
	Backend    string   `json:"backend"`
	StartedAt  string   `json:"started_at"`
	Domains    []string `json:"domains"`
	LogFile    string   `json:"log_file,omitempty"`
	Uptime     string   `json:"uptime"`
	Status     string   `json:"status"`
}

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
				if jsonOutput {
					return json.NewEncoder(os.Stdout).Encode([]statusMount{})
				}
				fmt.Println("No active mounts.")
				return nil
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(statusMounts(mounts))
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

func statusMounts(mounts []daemon.PIDInfo) []statusMount {
	items := make([]statusMount, 0, len(mounts))
	for _, mount := range mounts {
		items = append(items, statusMount{
			PID:        mount.PID,
			Mountpoint: mount.Mountpoint,
			Backend:    mount.Backend,
			StartedAt:  mount.StartedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
			Domains:    mount.Domains,
			LogFile:    mount.LogFile,
			Uptime:     mount.Uptime(),
			Status:     mount.Status(),
		})
	}
	return items
}
