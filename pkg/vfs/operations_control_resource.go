package vfs

import (
	"strings"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

func (o *Operations) ensureResourceControlFiles(parent, child *VNode) {
	if parent == nil || child == nil || !isReplaceableDriveFile(child) {
		return
	}

	targetPath := child.Path()
	requestName := replaceRequestName(child.Name)
	resultName := replaceResultName(child.Name)

	if parent.GetChild(requestName) == nil {
		parent.AddChild(newTargetedControlFile(parent, requestName, ControlRequestFile, "replace", targetPath))
	}
	if parent.GetChild(resultName) == nil {
		parent.AddChild(newTargetedControlFile(parent, resultName, ControlResultFile, "replace", targetPath))
	}
}

func (o *Operations) removeResourceControlFiles(parent, child *VNode) {
	if parent == nil || child == nil {
		return
	}

	requestName := replaceRequestName(child.Name)
	resultName := replaceResultName(child.Name)

	parent.mu.Lock()
	delete(parent.children, requestName)
	delete(parent.children, resultName)
	parent.mu.Unlock()

	o.controls.Delete(pathJoin(parent.Path(), requestName))
	o.controls.Delete(pathJoin(parent.Path(), resultName))
}

func isReplaceableDriveFile(node *VNode) bool {
	if node == nil {
		return false
	}
	return node.Kind == NodeKindResource &&
		node.Domain == "drive" &&
		node.NodeType == NodeFile &&
		node.DocType == doctype.TypeFile &&
		!strings.Contains(node.Token, "|")
}

func replaceRequestName(resourceName string) string {
	return resourceName + "._replace.request.json"
}

func replaceResultName(resourceName string) string {
	return resourceName + "._replace.result.json"
}
