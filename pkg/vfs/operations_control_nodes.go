package vfs

import (
	"encoding/json"
	"fmt"
)

func (o *Operations) ensureControlChildren(node *VNode) {
	if node == nil || !node.IsDir() || node.Kind != NodeKindResource {
		return
	}
	if node.GetChild("_meta") == nil || node.GetChild("_ops") == nil || node.GetChild("_queries") == nil || node.GetChild("_views") == nil {
		addControlNodes(node, node.Path())
	}
}

func (o *Operations) listControlDir(node *VNode) ([]*VNode, error) {
	node.ClearChildren()
	switch node.Control {
	case ControlMetaDir:
		node.AddChild(newControlFile(node, "index.json", ControlIndexFile, "index"))
		node.AddChild(newControlFile(node, "capabilities.json", ControlCapsFile, "capabilities"))
	case ControlOpsDir:
		node.AddChild(newControlFile(node, "exec.request.json", ControlRequestFile, "exec"))
		node.AddChild(newControlFile(node, "exec.result.json", ControlResultFile, "exec"))
		for _, action := range opActions(node.Domain) {
			node.AddChild(newControlFile(node, action+".request.json", ControlRequestFile, action))
			node.AddChild(newControlFile(node, action+".result.json", ControlResultFile, action))
		}
	case ControlQueriesDir:
		for _, action := range queryActions(node.Domain) {
			node.AddChild(newControlFile(node, action+".request.json", ControlRequestFile, action))
			node.AddChild(newControlFile(node, action+".result.json", ControlResultFile, action))
		}
	case ControlViewsDir:
		for _, action := range queryActions(node.Domain) {
			viewDir := newControlDir(node, action, ControlViewDir, action)
			viewDir.AddChild(newControlFile(viewDir, "results.json", ControlViewFile, action))
			node.AddChild(viewDir)
		}
	case ControlViewDir:
		node.AddChild(newControlFile(node, "results.json", ControlViewFile, node.Action))
	}
	node.SetPopulated()
	return node.Children(), nil
}

func (o *Operations) readControlNode(node *VNode) ([]byte, error) {
	switch node.Control {
	case ControlIndexFile:
		target := o.tree.Resolve(node.TargetPath)
		if target == nil {
			target = &VNode{Kind: NodeKindResource}
		}
		payload := map[string]any{
			"path":        node.TargetPath,
			"domain":      node.Domain,
			"kind":        target.Kind,
			"node_type":   target.NodeType,
			"has_more":    target.Page.HasMore,
			"next_cursor": target.Page.NextCursor,
			"window_size": target.Page.WindowSize,
			"sort_key":    target.Page.SortKey,
			"truncated":   target.Page.Truncated,
		}
		return json.MarshalIndent(payload, "", "  ")
	case ControlCapsFile:
		payload := map[string]any{
			"domain":  node.Domain,
			"queries": queryActions(node.Domain),
			"ops":     append([]string{"exec"}, opActions(node.Domain)...),
		}
		return json.MarshalIndent(payload, "", "  ")
	case ControlRequestFile, ControlResultFile, ControlViewFile:
		if data := o.controls.Get(node.Path()); data != nil {
			return data, nil
		}
		if node.Control == ControlRequestFile {
			return o.requestTemplate(node)
		}
		if node.Control == ControlViewFile {
			return []byte("{}\n"), nil
		}
		return []byte{}, nil
	}
	return nil, fmt.Errorf("unsupported control read: %s", node.Path())
}

func (o *Operations) requestTemplate(node *VNode) ([]byte, error) {
	payload := map[string]any{
		"domain":      node.Domain,
		"action":      node.Action,
		"target_path": node.TargetPath,
		"query":       "",
		"flags":       map[string]any{},
		"params":      map[string]any{},
		"data":        map[string]any{},
		"args":        []string{},
	}
	if node.Domain == "drive" && node.Action == "replace" {
		payload["flags"] = map[string]any{"file_path": ""}
		payload["data"] = map[string]any{"content": "", "content_base64": ""}
		payload["help"] = "Replace an existing Drive file by setting target_path to the file path and providing either flags.file_path or data.content_base64/data.content."
	} else if node.Action == "exec" {
		payload["help"] = "Set args to exact lark-cli arguments, for example [\"schema\", \"drive.files.list\"]. Non-_system domains auto-prefix the domain when args do not start with it."
	} else if isQueryNode(node) {
		if spec, ok := querySpec(node.Domain, node.Action); ok {
			payload["base_args"] = spec.args
			payload["help"] = "Set query for shortcut searches, or flags/params/data for command-specific arguments."
		}
	} else if spec, ok := actionSpecFor(node.Domain, node.Action); ok {
		payload["base_args"] = spec.args
		payload["help"] = "Set flags for CLI flags, params for --params JSON, data for --data JSON, or args to override the base command completely."
	}
	return json.MarshalIndent(payload, "", "  ")
}

func isQueryNode(node *VNode) bool {
	parent := node.Parent()
	return parent != nil && parent.Control == ControlQueriesDir
}
