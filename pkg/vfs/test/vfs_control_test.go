package vfs_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/vfs"
	"github.com/IchenDEV/larkfs/tests/testutil"
)

func TestVFSControlActionsAcrossDomainsBlackbox(t *testing.T) {
	domains := []string{"approval", "base", "calendar", "contact", "docs", "drive", "im", "mail", "meetings", "minutes", "sheets", "tasks", "vc", "wiki", "_system"}
	runner := &testutil.Runner{Out: []byte(`{"ok":true}`)}
	ops := vfs.NewOperations(vfs.OperationsConfig{
		CLI:  runner,
		Tree: vfs.NewTree(domains),
		TTL:  time.Minute,
	})

	for _, domain := range domains {
		if _, err := ops.ReadDir(context.Background(), "/"+domain+"/_meta"); err != nil {
			t.Fatalf("ReadDir(%s meta) error: %v", domain, err)
		}
		if domain == "approval" || domain == "base" || domain == "contact" || domain == "docs" || domain == "minutes" || domain == "sheets" || domain == "vc" || domain == "_system" {
			if entries, err := ops.ReadDir(context.Background(), "/"+domain); err != nil || len(entries) == 0 {
				t.Fatalf("ReadDir(%s root) = %+v, %v", domain, entries, err)
			}
		}
		if _, err := ops.Read(context.Background(), "/"+domain+"/_meta/index.json"); err != nil {
			t.Fatalf("Read(%s index) error: %v", domain, err)
		}
		caps, err := ops.Read(context.Background(), "/"+domain+"/_meta/capabilities.json")
		if err != nil || !strings.Contains(string(caps), "queries") {
			t.Fatalf("Read(%s caps) = %s, %v", domain, caps, err)
		}

		queryNodes, err := ops.ReadDir(context.Background(), "/"+domain+"/_queries")
		if err != nil {
			t.Fatalf("ReadDir(%s queries) error: %v", domain, err)
		}
		for _, node := range queryNodes {
			if !strings.HasSuffix(node.Name, ".request.json") {
				continue
			}
			path := "/" + domain + "/_queries/" + node.Name
			template, err := ops.Read(context.Background(), path)
			if err != nil || !strings.Contains(string(template), "target_path") {
				t.Fatalf("Read(%s template) = %s, %v", path, template, err)
			}
			if _, err := ops.RunQuery(context.Background(), path, []byte(`{"query":"x","flags":{"token":"tok"},"params":{"page_size":1},"data":{"sample":true}}`)); err != nil {
				t.Fatalf("RunQuery(%s) error: %v", path, err)
			}
		}

		opNodes, err := ops.ReadDir(context.Background(), "/"+domain+"/_ops")
		if err != nil {
			t.Fatalf("ReadDir(%s ops) error: %v", domain, err)
		}
		for _, node := range opNodes {
			if !strings.HasSuffix(node.Name, ".request.json") {
				continue
			}
			path := "/" + domain + "/_ops/" + node.Name
			template, err := ops.Read(context.Background(), path)
			if err != nil || !strings.Contains(string(template), "target_path") {
				t.Fatalf("Read(%s template) = %s, %v", path, template, err)
			}
			if strings.HasSuffix(node.Name, "exec.request.json") {
				if _, err := ops.ExecuteOp(context.Background(), path, []byte(`{"args":["schema","drive.files.list"]}`)); err != nil {
					t.Fatalf("ExecuteOp(%s exec) error: %v", path, err)
				}
				continue
			}
			result, err := ops.ExecuteOp(context.Background(), path, []byte(`{"flags":{"token":"tok"},"params":{"page_size":1},"data":{"sample":true}}`))
			if domain == "meetings" && err != nil {
				continue
			}
			if err != nil || len(result) == 0 {
				t.Fatalf("ExecuteOp(%s) = %s, %v", path, result, err)
			}
		}
	}
}
