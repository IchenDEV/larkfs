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

func newServeCmd() *cobra.Command {
	var cfg config.ServeConfig

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve Lark filesystem via WebDAV",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Daemon {
				return daemon.ForkDaemon(fmt.Sprintf("webdav:%d", cfg.Port), os.Args)
			}
			return runServe(cfg)
		},
	}

	f := cmd.Flags()
	f.IntVar(&cfg.Port, "port", 8080, "WebDAV server port")
	f.StringVar(&cfg.Addr, "addr", "localhost", "Bind address")
	f.BoolVarP(&cfg.Daemon, "daemon", "d", false, "Run as background daemon")
	f.StringVar(&cfg.LogLevel, "log-level", "info", "Log level")
	f.BoolVar(&cfg.ReadOnly, "read-only", false, "Serve in read-only mode")
	f.StringVar(&cfg.Domains, "domains", "drive,wiki,im,calendar,tasks,mail,meetings,approval,base,contact,docs,minutes,sheets,vc,_system", "Enabled domains")
	f.StringVar(&cfg.LarkCLIPath, "lark-cli", "", "Path to lark-cli binary")

	return cmd
}

func runServe(cfg config.ServeConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port)
	slog.Info("starting WebDAV server", "addr", addr)

	srv, err := mount.NewWebDAVServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create WebDAV server: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down WebDAV server...")
		srv.Close()
	}()

	slog.Info("WebDAV server listening", "addr", addr)
	return srv.Serve(addr)
}
