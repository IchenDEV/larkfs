package daemon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IchenDEV/larkfs/pkg/config"
)

type PIDInfo struct {
	PID        int       `json:"pid"`
	Mountpoint string    `json:"mountpoint"`
	Backend    string    `json:"backend"`
	StartedAt  time.Time `json:"started_at"`
	Domains    []string  `json:"domains"`
	LogFile    string    `json:"log_file,omitempty"`
}

func (p PIDInfo) Uptime() string {
	d := time.Since(p.StartedAt)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func (p PIDInfo) Status() string {
	if isProcessAlive(p.PID) {
		return "healthy"
	}
	return "stale"
}

func pidFilePath(mountpoint string) string {
	h := sha256.Sum256([]byte(mountpoint))
	name := hex.EncodeToString(h[:8]) + ".pid"
	return filepath.Join(config.MountsDir(), name)
}

func WritePIDFile(mountpoint, backend string, domains []string) error {
	info := PIDInfo{
		PID:        os.Getpid(),
		Mountpoint: mountpoint,
		Backend:    backend,
		StartedAt:  time.Now(),
		Domains:    domains,
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(config.MountsDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(pidFilePath(mountpoint), data, 0o644)
}

func RemovePIDFile(mountpoint string) {
	os.Remove(pidFilePath(mountpoint))
}

func ReadPIDFile(path string) (*PIDInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info PIDInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
