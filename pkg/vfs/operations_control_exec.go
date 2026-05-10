package vfs

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

func (o *Operations) writeControlNode(ctx context.Context, node *VNode, data []byte) error {
	o.controls.Set(node.Path(), data)
	node.SetModTime(time.Now())

	switch {
	case strings.HasSuffix(node.Name, "exec.request.json"):
		out, err := o.executeControlExec(ctx, node, data)
		if err != nil {
			return err
		}
		o.storeSiblingResult(node, out)
		return nil
	case strings.HasSuffix(node.Name, ".request.json"):
		var (
			out []byte
			err error
		)
		if isQueryNode(node) {
			out, err = o.executeQuery(ctx, node, data)
		} else {
			out, err = o.executeAction(ctx, node, data)
		}
		if err != nil {
			return err
		}
		o.storeSiblingResult(node, out)
		if isQueryNode(node) {
			o.storeViewResult(node, out)
		}
		return nil
	default:
		return fmt.Errorf("control node is not writable: %s", node.Path())
	}
}

func (o *Operations) storeSiblingResult(node *VNode, data []byte) {
	resultPath := strings.Replace(node.Path(), ".request.json", ".result.json", 1)
	o.controls.Set(resultPath, data)
}

func (o *Operations) storeViewResult(node *VNode, data []byte) {
	viewPath := path.Join("/", node.Domain, "_views", node.Action, "results.json")
	o.controls.Set(viewPath, data)
}

var blockedCommands = map[string]bool{
	"auth logout":  true,
	"auth revoke":  true,
	"config init":  true,
	"config reset": true,
}

func (o *Operations) executeControlExec(ctx context.Context, node *VNode, data []byte) ([]byte, error) {
	var req execRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse exec request: %w", err)
	}
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("exec request requires args")
	}
	args := req.Args
	if node.Domain != "_system" && !hasDomainPrefix(node.Domain, args) {
		args = append([]string{node.Domain}, args...)
	}
	if isBlockedCommand(args) {
		return nil, fmt.Errorf("%w: command %q is blocked for safety", ErrUnsupported, strings.Join(args, " "))
	}
	return o.cli.Run(ctx, args...)
}

func isBlockedCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}
	key := args[0] + " " + args[1]
	return blockedCommands[key]
}

func hasDomainPrefix(domain string, args []string) bool {
	if len(args) == 0 {
		return false
	}
	return args[0] == domain
}

func (o *Operations) executeQuery(ctx context.Context, node *VNode, data []byte) ([]byte, error) {
	var req execRequest
	if len(data) > 0 {
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("parse query request: %w", err)
		}
	}
	if len(req.Args) > 0 {
		return o.executeControlExec(ctx, node, data)
	}

	spec, ok := querySpec(node.Domain, node.Action)
	if !ok {
		return nil, fmt.Errorf("unsupported query: %s/%s", node.Domain, node.Action)
	}
	args := append([]string(nil), spec.args...)
	if spec.queryArg != "" && req.Query != "" {
		args = append(args, spec.queryArg, req.Query)
		req.Query = ""
	}
	if spec.pageAll {
		args = append(args, "--format", "json", "--page-all", "--page-limit", "0")
	}
	args = appendRequestArgs(args, req)

	out, err := o.cli.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var pretty any
	if err := json.Unmarshal(out, &pretty); err == nil {
		return json.MarshalIndent(pretty, "", "  ")
	}
	return out, nil
}

func (o *Operations) executeAction(ctx context.Context, node *VNode, data []byte) ([]byte, error) {
	var req execRequest
	if len(data) > 0 {
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("parse action request: %w", err)
		}
	}
	if node.Domain == "drive" && node.Action == "replace" {
		return o.executeDriveReplace(ctx, node, req)
	}
	if len(req.Args) > 0 {
		return o.executeControlExec(ctx, node, data)
	}

	spec, ok := actionSpecFor(node.Domain, node.Action)
	if !ok {
		return nil, fmt.Errorf("unsupported action: %s/%s", node.Domain, node.Action)
	}
	args := append([]string(nil), spec.args...)
	args = appendRequestArgs(args, req)

	out, err := o.cli.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var pretty any
	if err := json.Unmarshal(out, &pretty); err == nil {
		return json.MarshalIndent(pretty, "", "  ")
	}
	return out, nil
}

func appendRequestArgs(args []string, req execRequest) []string {
	if req.Query != "" {
		args = append(args, "--query", req.Query)
	}
	if len(req.Params) > 0 {
		raw, _ := json.Marshal(req.Params)
		args = append(args, "--params", string(raw))
	}
	if len(req.Data) > 0 {
		raw, _ := json.Marshal(req.Data)
		args = append(args, "--data", string(raw))
	}

	keys := make([]string, 0, len(req.Flags))
	for k := range req.Flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := req.Flags[k]
		flag := "--" + strings.ReplaceAll(k, "_", "-")
		switch typed := v.(type) {
		case bool:
			if typed {
				args = append(args, flag)
			}
		default:
			args = append(args, flag, fmt.Sprint(typed))
		}
	}
	return args
}
