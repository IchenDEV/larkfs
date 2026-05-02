package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type doctorCLIStatus struct {
	Found bool   `json:"found"`
	Path  string `json:"path,omitempty"`
	Error string `json:"error,omitempty"`
}

type doctorAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	UserName      string `json:"user_name,omitempty"`
	Identity      string `json:"identity,omitempty"`
	Error         string `json:"error,omitempty"`
}

type doctorReport struct {
	OK      bool             `json:"ok"`
	LarkCLI doctorCLIStatus  `json:"lark_cli"`
	Auth    doctorAuthStatus `json:"auth"`
	Checks  []doctorCheck    `json:"checks"`
}

func newDoctorCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system dependencies and connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runDoctor(jsonOutput bool) error {
	report := collectDoctorReport()

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
		return nil
	}

	for _, check := range report.Checks {
		printCheck(check.OK, check.Message)
	}
	if !report.OK {
		return fmt.Errorf("some checks failed")
	}
	return nil
}

func collectDoctorReport() doctorReport {
	report := doctorReport{}
	path, err := cli.FindLarkCLI("")
	if err != nil {
		report.LarkCLI = doctorCLIStatus{
			Found: false,
			Error: err.Error(),
		}
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "lark_cli",
			Status:  "fail",
			OK:      false,
			Message: "lark-cli: not found",
		})
	} else {
		report.LarkCLI = doctorCLIStatus{
			Found: true,
			Path:  path,
		}
		report.Checks = append(report.Checks, doctorCheck{
			Name:    "lark_cli",
			Status:  "pass",
			OK:      true,
			Message: fmt.Sprintf("lark-cli found: %s", path),
		})
	}

	if path != "" {
		out, err := exec.Command(path, "auth", "status").CombinedOutput()
		if err != nil {
			report.Auth = doctorAuthStatus{
				Authenticated: false,
				Error:         strings.TrimSpace(string(out)),
			}
			report.Checks = append(report.Checks, doctorCheck{
				Name:    "auth",
				Status:  "fail",
				OK:      false,
				Message: fmt.Sprintf("lark-cli auth: not logged in (%s)", strings.TrimSpace(string(out))),
			})
		} else {
			var authInfo struct {
				UserName string `json:"userName"`
				Identity string `json:"identity"`
			}
			if json.Unmarshal(out, &authInfo) == nil && authInfo.UserName != "" {
				report.Auth = doctorAuthStatus{
					Authenticated: true,
					UserName:      authInfo.UserName,
					Identity:      authInfo.Identity,
				}
				report.Checks = append(report.Checks, doctorCheck{
					Name:    "auth",
					Status:  "pass",
					OK:      true,
					Message: fmt.Sprintf("lark-cli auth: logged in as %s (%s)", authInfo.UserName, authInfo.Identity),
				})
			} else {
				report.Auth = doctorAuthStatus{Authenticated: true}
				report.Checks = append(report.Checks, doctorCheck{
					Name:    "auth",
					Status:  "pass",
					OK:      true,
					Message: "lark-cli auth: authenticated",
				})
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
					report.Checks = append(report.Checks, doctorCheck{
						Name:    c.Name,
						Status:  c.Status,
						OK:      ok,
						Message: msg,
						Hint:    c.Hint,
					})
				}
			}
		}
	}

	if fuseCheck, ok := checkFUSE(); ok {
		report.Checks = append(report.Checks, fuseCheck)
	}

	report.OK = true
	for _, check := range report.Checks {
		if !check.OK {
			report.OK = false
			break
		}
	}
	return report
}

func checkFUSE() (doctorCheck, bool) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("mount_macfuse"); err == nil {
			return doctorCheck{Name: "fuse", Status: "pass", OK: true, Message: "FUSE available: macFUSE"}, true
		} else if _, err := exec.LookPath("mount_fuse-t"); err == nil {
			return doctorCheck{Name: "fuse", Status: "pass", OK: true, Message: "FUSE available: Fuse-T"}, true
		} else {
			return doctorCheck{Name: "fuse", Status: "fail", OK: false, Message: "FUSE not available (install macfuse or fuse-t)"}, true
		}
	case "linux":
		if _, err := exec.LookPath("fusermount3"); err == nil {
			return doctorCheck{Name: "fuse", Status: "pass", OK: true, Message: "FUSE available: fuse3"}, true
		} else if _, err := exec.LookPath("fusermount"); err == nil {
			return doctorCheck{Name: "fuse", Status: "pass", OK: true, Message: "FUSE available: fuse"}, true
		} else {
			return doctorCheck{Name: "fuse", Status: "fail", OK: false, Message: "FUSE not available (install fuse3)"}, true
		}
	}
	return doctorCheck{}, false
}

func printCheck(ok bool, msg string) {
	if ok {
		fmt.Printf("[✓] %s\n", msg)
	} else {
		fmt.Printf("[✗] %s\n", msg)
	}
}
