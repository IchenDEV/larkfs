package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/mount"
	"github.com/IchenDEV/larkfs/pkg/vfs"
	"github.com/spf13/cobra"
)

type nativeItem struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent_id,omitempty"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	DocType     string `json:"doc_type,omitempty"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
	Version     string `json:"version"`
}

func newNativeCmd() *cobra.Command {
	var cfg config.MountConfig

	cmd := &cobra.Command{
		Use:    "native",
		Short:  "File Provider bridge commands",
		Hidden: true,
	}

	f := cmd.PersistentFlags()
	f.StringVar(&cfg.CacheDir, "cache-dir", "", "Cache directory (default: ~/.larkfs/cache)")
	f.StringVar(&cfg.CacheSize, "cache-size", config.DefaultCacheSize, "Cache size limit")
	f.IntVar(&cfg.MetadataTTL, "metadata-ttl", 60, "Metadata cache TTL in seconds")
	f.BoolVar(&cfg.ReadOnly, "read-only", true, "Expose the native bridge as read-only")
	f.StringVar(&cfg.Domains, "domains", config.DefaultDomainsValue, "Enabled domains (comma-separated)")
	f.StringVar(&cfg.LarkCLIPath, "lark-cli", "", "Path to lark-cli binary (auto-detect)")

	cmd.AddCommand(
		newNativeItemCmd(&cfg),
		newNativeListCmd(&cfg),
		newNativeFetchCmd(&cfg),
	)
	return cmd
}

func newNativeItemCmd(cfg *config.MountConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "item <path>",
		Short: "Return File Provider metadata for one LarkFS path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ops, err := nativeOps(*cfg)
			if err != nil {
				return err
			}
			item, err := nativeItemForPath(cmd.Context(), ops, args[0])
			if err != nil {
				return err
			}
			return writeNativeJSON(item)
		},
	}
}

func newNativeListCmd(cfg *config.MountConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list <path>",
		Short: "Return File Provider children for one LarkFS directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ops, err := nativeOps(*cfg)
			if err != nil {
				return err
			}
			children, err := ops.ReadDir(cmd.Context(), cleanNativePath(args[0]))
			if err != nil {
				return err
			}
			items := make([]nativeItem, 0, len(children))
			for _, child := range children {
				items = append(items, nativeItemFromNode(child.Path(), child))
			}
			sort.Slice(items, func(i, j int) bool {
				if items[i].Kind != items[j].Kind {
					return items[i].Kind == "directory"
				}
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			})
			return writeNativeJSON(items)
		},
	}
}

func newNativeFetchCmd(cfg *config.MountConfig) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "fetch <path>",
		Short: "Fetch File Provider contents for one LarkFS file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ops, err := nativeOps(*cfg)
			if err != nil {
				return err
			}
			data, err := ops.Read(cmd.Context(), cleanNativePath(args[0]))
			if err != nil {
				return err
			}
			if output == "" {
				_, err = os.Stdout.Write(data)
				return err
			}
			return os.WriteFile(output, data, 0o644)
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "Write contents to this path instead of stdout")
	return cmd
}

func nativeOps(cfg config.MountConfig) (*vfs.Operations, error) {
	if err := cfg.Resolve(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return mount.NewOperations(cfg)
}

func nativeItemForPath(ctx context.Context, ops *vfs.Operations, rawPath string) (nativeItem, error) {
	itemPath := cleanNativePath(rawPath)
	if itemPath == "/" {
		return nativeRootItem(), nil
	}
	node, err := ops.Stat(ctx, itemPath)
	if err != nil {
		return nativeItem{}, err
	}
	return nativeItemFromNode(itemPath, node), nil
}

func nativeRootItem() nativeItem {
	return nativeItem{
		ID:          nativePathID("/"),
		Path:        "/",
		Name:        "LarkFS",
		Kind:        "directory",
		ContentType: "public.folder",
		Version:     "root",
	}
}

func nativeItemFromNode(itemPath string, node *vfs.VNode) nativeItem {
	itemPath = cleanNativePath(itemPath)
	kind := "file"
	if node.IsDir() {
		kind = "directory"
	}
	return nativeItem{
		ID:          nativePathID(itemPath),
		ParentID:    nativePathID(path.Dir(itemPath)),
		Path:        itemPath,
		Name:        nativeName(itemPath, node),
		Kind:        kind,
		DocType:     string(node.DocType),
		ContentType: nativeContentType(node),
		Size:        node.GetSize(),
		CreatedAt:   nativeTime(node.CreatedTime),
		ModifiedAt:  nativeTime(node.GetModTime()),
		Version:     nativeVersion(itemPath, node),
	}
}

func nativeName(itemPath string, node *vfs.VNode) string {
	if itemPath == "/" {
		return "LarkFS"
	}
	if node.Name != "" {
		return node.Name
	}
	return path.Base(itemPath)
}

func nativeContentType(node *vfs.VNode) string {
	if node.IsDir() {
		return "public.folder"
	}
	switch {
	case strings.HasSuffix(node.Name, ".md"):
		return "net.daringfireball.markdown"
	case strings.HasSuffix(node.Name, ".csv"):
		return "public.comma-separated-values-text"
	case strings.HasSuffix(node.Name, ".json"):
		return "public.json"
	case strings.HasSuffix(node.Name, ".jsonl"):
		return "public.json"
	case strings.HasSuffix(node.Name, ".txt"):
		return "public.plain-text"
	case strings.HasSuffix(node.Name, ".mp4"):
		return "public.mpeg-4"
	case strings.HasSuffix(node.Name, ".png"):
		return "public.png"
	case node.DocType == doctype.TypeFile:
		return "public.data"
	default:
		return "public.data"
	}
}

func nativeVersion(itemPath string, node *vfs.VNode) string {
	if mt := node.GetModTime(); !mt.IsZero() {
		return fmt.Sprintf("%d", mt.UnixNano())
	}
	return nativePathID(itemPath)
}

func nativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func nativePathID(rawPath string) string {
	clean := cleanNativePath(rawPath)
	if clean == "/" {
		return "root"
	}
	return "path." + base64.RawURLEncoding.EncodeToString([]byte(clean))
}

func cleanNativePath(rawPath string) string {
	clean := path.Clean("/" + strings.TrimPrefix(rawPath, "/"))
	if clean == "." {
		return "/"
	}
	return clean
}

func writeNativeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
