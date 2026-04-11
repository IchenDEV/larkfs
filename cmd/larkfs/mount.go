package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/daemon"
	"github.com/IchenDEV/larkfs/pkg/mount"
	"github.com/spf13/cobra"
)

func newMountCmd() *cobra.Command {
	var cfg config.MountConfig

	cmd := &cobra.Command{
		Use:   "mount <mountpoint>",
		Short: "Mount Lark filesystem via FUSE",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Mountpoint = args[0]

			if cfg.Daemon && !daemon.IsDaemonChild() {
				return daemon.ForkDaemon(cfg.Mountpoint, os.Args)
			}

			return runMount(cfg)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&cfg.Daemon, "daemon", "d", false, "Run as background daemon")
	f.StringVar(&cfg.CacheDir, "cache-dir", "", "Cache directory (default: ~/.larkfs/cache)")
	f.StringVar(&cfg.CacheSize, "cache-size", "500MB", "Cache size limit")
	f.IntVar(&cfg.MetadataTTL, "metadata-ttl", 60, "Metadata cache TTL in seconds")
	f.StringVar(&cfg.LogFile, "log-file", "", "Log file path (default: ~/.larkfs/larkfs.log)")
	f.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug/info/warn/error)")
	f.BoolVar(&cfg.ReadOnly, "read-only", false, "Mount in read-only mode")
	f.StringVar(&cfg.Domains, "domains", "drive,wiki,im,calendar,tasks,mail,meetings,approval,base,contact,docs,minutes,sheets,vc,_system", "Enabled domains (comma-separated)")
	f.StringVar(&cfg.LarkCLIPath, "lark-cli", "", "Path to lark-cli binary (auto-detect)")

	return cmd
}

func runMount(cfg config.MountConfig) error {
	if err := cfg.Resolve(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := daemon.CleanStaleMounts(); err != nil {
		slog.Warn("failed to clean stale mounts", "error", err)
	}

	if err := daemon.CheckExistingMount(cfg.Mountpoint); err != nil {
		return err
	}

	slog.Info("mounting larkfs", "mountpoint", cfg.Mountpoint, "domains", cfg.Domains, "read-only", cfg.ReadOnly)

	srv, err := mount.NewFUSEServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create FUSE server: %w", err)
	}

	if err := daemon.WritePIDFile(cfg.Mountpoint, "fuse", cfg.EnabledDomains()); err != nil {
		slog.Warn("failed to write PID file", "error", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal, unmounting...")
		srv.Unmount()
		daemon.RemovePIDFile(cfg.Mountpoint)
	}()

	slog.Info("larkfs mounted", "mountpoint", cfg.Mountpoint)
	srv.Wait()
	slog.Info("larkfs shutdown complete")
	return nil
}
