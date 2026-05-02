package vfs

import (
	"path"
	"strings"
)

func (o *Operations) domainFromPath(node *VNode) string {
	for cur := node; cur != nil; cur = cur.parent {
		if cur.Domain != "" {
			return cur.Domain
		}
	}
	return ""
}

func pathJoin(parent, child string) string {
	if parent == "/" {
		return "/" + child
	}
	return parent + "/" + child
}

func pathBase(p string) string {
	clean := path.Clean("/" + strings.TrimPrefix(p, "/"))
	if clean == "/" {
		return ""
	}
	return path.Base(clean)
}
