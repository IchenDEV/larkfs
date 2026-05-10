package vfs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

func (o *Operations) executeDriveReplace(ctx context.Context, node *VNode, req execRequest) ([]byte, error) {
	if o.drive == nil {
		return nil, fmt.Errorf("%w: drive adapter not configured", ErrUnsupported)
	}

	if filePath, ok := requestString(req.Flags, "file_path"); ok {
		if err := validateFilePath(filePath, o.cacheDir); err != nil {
			return nil, err
		}
	}

	targetPath := req.TargetPath
	if targetPath == "" {
		targetPath = node.TargetPath
	}
	target, err := o.resolveNode(ctx, targetPath)
	if err != nil {
		return nil, err
	}
	if target.Kind != NodeKindResource || target.NodeType != NodeFile {
		return nil, fmt.Errorf("%w: replace target must be a file resource", ErrUnsupported)
	}
	if target.Domain != "drive" || target.DocType != doctype.TypeFile || strings.Contains(target.Token, "|") {
		return nil, fmt.Errorf("%w: replace only supports ordinary drive files", ErrUnsupported)
	}

	data, err := replacementContent(req)
	if err != nil {
		return nil, err
	}

	oldToken := target.Token
	if target.PendingCreate {
		if err := o.writeContent(ctx, target, data); err != nil {
			return nil, err
		}
	} else {
		parent := target.Parent()
		if parent == nil {
			return nil, fmt.Errorf("%w: cannot replace root-level target", ErrUnsupported)
		}
		newToken, err := o.drive.ReplaceFile(ctx, parent.Token, target.Token, target.Name, data)
		if err != nil {
			return nil, err
		}
		target.Token = newToken
		target.PendingCreate = false
		target.SetSize(int64(len(data)))
		target.SetModTime(time.Now())
	}

	result, err := json.MarshalIndent(map[string]any{
		"ok":            true,
		"path":          target.Path(),
		"old_token":     oldToken,
		"new_token":     target.Token,
		"token_changed": oldToken != target.Token,
		"size":          len(data),
		"mode":          "upload_delete",
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func replacementContent(req execRequest) ([]byte, error) {
	if filePath, ok := requestString(req.Flags, "file_path"); ok {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read replacement file: %w", err)
		}
		return data, nil
	}
	if encoded, ok := requestString(req.Data, "content_base64"); ok {
		data, err := decodeBase64(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode content_base64: %w", err)
		}
		return data, nil
	}
	if content, ok := requestString(req.Data, "content"); ok {
		return []byte(content), nil
	}
	return nil, fmt.Errorf("replace request requires flags.file_path or data.content_base64/data.content")
}

func requestString(values map[string]any, key string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}
	raw, ok := values[key]
	if !ok {
		return "", false
	}
	text, ok := raw.(string)
	if !ok || text == "" {
		return "", false
	}
	return text, true
}

func decodeBase64(value string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return data, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}

func validateFilePath(filePath, cacheDir string) error {
	if cacheDir == "" {
		return fmt.Errorf("%w: file_path requires cache-dir to be configured", ErrUnsupported)
	}
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolve file_path: %w", err)
	}
	absCache, err := filepath.Abs(cacheDir)
	if err != nil {
		return fmt.Errorf("resolve cache-dir: %w", err)
	}
	if !strings.HasPrefix(abs, absCache+string(filepath.Separator)) && abs != absCache {
		return fmt.Errorf("%w: file_path must be within cache directory %s", ErrUnsupported, absCache)
	}
	return nil
}
