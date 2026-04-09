package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize lark-cli configuration and authentication",
		Long:  "Set up lark-cli with app credentials and user login for LarkFS.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

func runInit() error {
	path, err := cli.FindLarkCLI("")
	if err != nil {
		fmt.Println("lark-cli is not installed.")
		fmt.Println("Install it with: npm install -g @larksuite/cli")
		return fmt.Errorf("lark-cli not found")
	}
	fmt.Printf("[✓] lark-cli found: %s\n", path)

	if !isConfigured(path) {
		fmt.Println("\nlark-cli is not configured. Starting setup...")
		fmt.Println("This will open a browser to create a new Lark app.")
		cmd := exec.Command(path, "config", "init", "--new")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("config init failed: %w", err)
		}
		fmt.Println("\n[✓] Configuration complete.")
	} else {
		fmt.Println("[✓] lark-cli already configured.")
	}

	if !isLoggedIn(path) {
		fmt.Println("\nNo user logged in. Starting OAuth login...")
		fmt.Println("This will open a browser for authorization.")
		cmd := exec.Command(path, "auth", "login", "--domain", "all")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("auth login failed: %w", err)
		}
		fmt.Println("\n[✓] Login complete.")
	} else {
		fmt.Println("[✓] User already logged in.")
	}

	fmt.Println("\nAll set! You can now use:")
	fmt.Println("  larkfs serve    - Start WebDAV server")
	fmt.Println("  larkfs mount    - Mount via FUSE (requires macFUSE/fuse-t)")
	fmt.Println("  larkfs doctor   - Check system status")
	return nil
}

func isConfigured(cliPath string) bool {
	out, err := exec.Command(cliPath, "config", "show").CombinedOutput()
	if err != nil {
		return false
	}
	var cfg struct {
		AppID string `json:"appId"`
	}
	if json.Unmarshal(out, &cfg) != nil {
		return false
	}
	return cfg.AppID != ""
}

func isLoggedIn(cliPath string) bool {
	out, err := exec.Command(cliPath, "auth", "status").CombinedOutput()
	if err != nil {
		return false
	}
	var status struct {
		TokenStatus string `json:"tokenStatus"`
	}
	if json.Unmarshal(out, &status) != nil {
		return false
	}
	return status.TokenStatus == "valid" || status.TokenStatus == "needs_refresh"
}
