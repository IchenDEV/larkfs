package vfs

import (
	"context"
	"fmt"
	"path"
	"strings"
)

func (o *Operations) resolveNode(ctx context.Context, nodePath string) (*VNode, error) {
	node := o.tree.Resolve(nodePath)
	if node != nil {
		return node, nil
	}

	clean := path.Clean("/" + strings.TrimPrefix(nodePath, "/"))
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	cur := ""
	for _, part := range parts[:len(parts)-1] {
		cur = cur + "/" + part
		if _, err := o.ReadDir(ctx, cur); err != nil {
			break
		}
	}
	if _, err := o.ReadDir(ctx, pathpkgDir(nodePath)); err == nil {
		node = o.tree.Resolve(nodePath)
	}
	if node == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, nodePath)
	}
	return node, nil
}

func pathpkgDir(p string) string {
	p = path.Clean("/" + strings.TrimPrefix(p, "/"))
	if p == "/" {
		return "/"
	}
	return path.Dir(p)
}
