package doctype

import (
	"context"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

func deleteDriveResource(ctx context.Context, exec cli.Runner, token string, dt DocType) error {
	_, err := exec.Run(ctx,
		"drive", "+delete",
		"--file-token", token,
		"--type", driveDeleteType(dt),
		"--yes")
	return err
}

func driveDeleteType(dt DocType) string {
	switch dt {
	case TypeBitable:
		return "bitable"
	case TypeDocx:
		return "docx"
	case TypeFile:
		return "file"
	case TypeFolder:
		return "folder"
	case TypeSheet:
		return "sheet"
	default:
		return string(dt)
	}
}
