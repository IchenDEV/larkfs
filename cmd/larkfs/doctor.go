package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system dependencies and connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	var hasError bool

	path, err := cli.FindLarkCLI("")
	if err != nil {
		printCheck(false, "lark-cli: not found")
		hasError = true
	} else {
		printCheck(true, fmt.Sprintf("lark-cli found: %s", path))
	}

	if path != "" {
		out, err := exec.Command(path, "auth", "status").CombinedOutput()
		if err != nil {
			printCheck(false, fmt.Sprintf("lark-cli auth: not logged in (%s)", strings.TrimSpace(string(out))))
			hasError = true
		} else {
			var authInfo struct {
				UserName string `json:"userName"`
				Identity string `json:"identity"`
			}
			if json.Unmarshal(out, &authInfo) == nil && authInfo.UserName != "" {
				printCheck(true, fmt.Sprintf("lark-cli auth: logged in as %s (%s)", authInfo.UserName, authInfo.Identity))
			} else {
				printCheck(true, "lark-cli auth: authenticated")
			}
		}
	}

	if path != "" {
		doctorOut, err := exec.Command(path, "doctor").CombinedOutput()
		if err == nil {
			var doctorResult struct {
				Checks []struct {
					Name    string `json:"name"`
					Status  string `json:"status"`
					Message string `json:"message"`
					Hint    string `json:"hint"`
				} `json:"checks"`
				OK bool `json:"ok"`
			}
			if json.Unmarshal(doctorOut, &doctorResult) == nil {
				for _, c := range doctorResult.Checks {
					if c.Name == "cli_version" || c.Name == "token_exists" || c.Name == "token_verified" {
						continue
					}
					ok := c.Status == "pass" || c.Status == "warn"
					msg := fmt.Sprintf("lark-cli %s: %s", c.Name, c.Message)
					if c.Hint != "" {
						msg += " (" + c.Hint + ")"
					}
					if !ok {
						hasError = true
					}
					printCheck(ok, msg)
				}
			}
		}
	}

	checkFUSE()

	if hasError {
		return fmt.Errorf("some checks failed")
	}
	return nil
}

func checkFUSE() {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("mount_macfuse"); err == nil {
			printCheck(true, "FUSE available: macFUSE")
		} else if _, err := exec.LookPath("mount_fuse-t"); err == nil {
			printCheck(true, "FUSE available: Fuse-T")
		} else {
			printCheck(false, "FUSE not available (install macfuse or fuse-t)")
		}
	case "linux":
		if _, err := exec.LookPath("fusermount3"); err == nil {
			printCheck(true, "FUSE available: fuse3")
		} else if _, err := exec.LookPath("fusermount"); err == nil {
			printCheck(true, "FUSE available: fuse")
		} else {
			printCheck(false, "FUSE not available (install fuse3)")
		}
	}
}

func printCheck(ok bool, msg string) {
	if ok {
		fmt.Printf("[✓] %s\n", msg)
	} else {
		fmt.Printf("[✗] %s\n", msg)
	}
}
