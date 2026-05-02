package vfs

import (
	"context"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

func createDocTypeForName(name string) doctype.DocType {
	switch {
	case strings.HasSuffix(name, doctype.FileExtension(doctype.TypeDocx)):
		return doctype.TypeDocx
	default:
		return doctype.TypeFile
	}
}

func driveRemoteName(localName string, dt doctype.DocType) string {
	ext := doctype.FileExtension(dt)
	if ext == "" {
		return localName
	}
	return strings.TrimSuffix(localName, ext)
}

func (o *Operations) executeDriveMove(ctx context.Context, node, newParent *VNode) error {
	_, err := o.cli.Run(
		ctx,
		"drive", "+move",
		"--file-token", node.Token,
		"--folder-token", newParent.Token,
		"--type", string(node.DocType),
	)
	return err
}

func (o *Operations) relocateNode(node, oldParent, newParent *VNode, newName string) error {
	if oldParent == nil || newParent == nil {
		return nil
	}

	o.removeResourceControlFiles(oldParent, node)
	oldParent.mu.Lock()
	delete(oldParent.children, node.Name)
	oldParent.mu.Unlock()

	node.Name = newName
	node.ModTime = time.Now()
	newParent.AddChild(node)
	o.ensureResourceControlFiles(newParent, node)
	updateSubtreePaths(node)
	return nil
}

func updateSubtreePaths(node *VNode) {
	if node == nil {
		return
	}

	if node.Kind == NodeKindResource {
		node.TargetPath = resourceTargetPath(node)
	} else if parent := node.Parent(); parent != nil {
		node.TargetPath = parent.TargetPath
		node.Domain = parent.Domain
	}

	for _, child := range node.Children() {
		updateSubtreePaths(child)
	}
}

func resourceTargetPath(node *VNode) string {
	if node == nil {
		return ""
	}
	return node.Path()
}
