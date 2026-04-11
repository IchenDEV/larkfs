package vfs

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
)

type execRequest struct {
	Args   []string       `json:"args,omitempty"`
	Flags  map[string]any `json:"flags,omitempty"`
	Params map[string]any `json:"params,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
	Query  string         `json:"query,omitempty"`
}

type actionSpec struct {
	args      []string
	queryArg  string
	pageAll   bool
	rawDomain bool
}

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

func queryActions(domain string) []string {
	switch domain {
	case "contact":
		return []string{"search-user", "get-user"}
	case "docs":
		return []string{"search", "fetch"}
	case "approval":
		return []string{"instances", "tasks"}
	case "base":
		return []string{"data-query", "table-list", "record-list", "field-list", "view-list", "dashboard-list", "workflow-list"}
	case "drive":
		return []string{"comments", "statistics", "view-records", "metas"}
	case "im":
		return []string{"chat-search", "messages-search", "chat-messages-list", "threads-messages-list"}
	case "mail":
		return []string{"triage", "thread", "message"}
	case "minutes":
		return []string{"get", "download"}
	case "vc", "meetings":
		return []string{"search", "notes", "recording"}
	case "calendar":
		return []string{"agenda", "freebusy", "suggestion"}
	case "tasks":
		return []string{"get-my-tasks"}
	case "wiki":
		return []string{"spaces", "nodes"}
	case "sheets":
		return []string{"info", "read", "find"}
	case "_system":
		return []string{"schema", "doctor"}
	default:
		return nil
	}
}

func opActions(domain string) []string {
	switch domain {
	case "drive":
		return []string{"upload", "download", "import", "export", "move", "delete", "add-comment", "task-result"}
	case "wiki":
		return []string{"node-create"}
	case "im":
		return []string{"chat-create", "chat-update", "messages-send", "messages-reply", "reactions", "pins", "messages-resources-download"}
	case "calendar":
		return []string{"create", "rsvp"}
	case "tasks":
		return []string{"create", "update", "assign", "comment", "complete", "reopen", "followers", "reminder", "tasklist-create", "tasklist-task-add", "subtask"}
	case "mail":
		return []string{"send", "draft-create", "draft-edit", "reply", "reply-all", "forward", "watch"}
	case "approval":
		return []string{"approve", "reject", "transfer", "comment"}
	case "base":
		return []string{"base-create", "base-copy", "table-create", "table-update", "table-delete", "record-upsert", "record-delete", "field-create", "field-update", "field-delete", "view-create", "view-delete", "dashboard-create", "workflow-create"}
	case "docs":
		return []string{"create", "update", "media-download", "media-insert", "media-preview", "whiteboard-update"}
	case "minutes":
		return []string{"download"}
	case "vc", "meetings":
		return []string{"notes", "recording"}
	case "sheets":
		return []string{"create", "append", "write", "write-image", "export"}
	case "contact":
		return []string{"search-user", "get-user"}
	case "_system":
		return []string{"api", "schema", "doctor", "auth", "config", "profile", "event-subscribe"}
	default:
		return nil
	}
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
	if node.Action == "exec" {
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

func (o *Operations) writeControlNode(ctx context.Context, node *VNode, data []byte) error {
	o.controls.Set(node.Path(), data)
	node.ModTime = time.Now()

	switch {
	case strings.HasSuffix(node.Name, "exec.request.json"):
		out, err := o.executeControlExec(ctx, node, data)
		if err != nil {
			return err
		}
		o.storeSiblingResult(node, out)
		return nil
	case strings.HasSuffix(node.Name, ".request.json"):
		var out []byte
		var err error
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

func isQueryNode(node *VNode) bool {
	parent := node.Parent()
	return parent != nil && parent.Control == ControlQueriesDir
}

func (o *Operations) storeSiblingResult(node *VNode, data []byte) {
	resultPath := strings.Replace(node.Path(), ".request.json", ".result.json", 1)
	o.controls.Set(resultPath, data)
}

func (o *Operations) storeViewResult(node *VNode, data []byte) {
	viewPath := path.Join("/", node.Domain, "_views", node.Action, "results.json")
	o.controls.Set(viewPath, data)
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
	return o.cli.Run(ctx, args...)
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

func querySpec(domain, action string) (actionSpec, bool) {
	specs := map[string]map[string]actionSpec{
		"approval": {
			"instances": {args: []string{"approval", "instances", "list"}, pageAll: true},
			"tasks":     {args: []string{"approval", "tasks", "list"}, pageAll: true},
		},
		"contact": {
			"search-user": {args: []string{"contact", "+search-user"}, queryArg: "--query"},
			"get-user":    {args: []string{"contact", "+get-user"}},
		},
		"docs": {
			"search": {args: []string{"docs", "+search"}, queryArg: "--query"},
			"fetch":  {args: []string{"docs", "+fetch"}},
		},
		"drive": {
			"comments":     {args: []string{"drive", "file.comments", "list"}, pageAll: true},
			"statistics":   {args: []string{"drive", "file.statistics", "get"}},
			"view-records": {args: []string{"drive", "file.view_records", "list"}, pageAll: true},
			"metas":        {args: []string{"drive", "metas", "batch_query"}},
		},
		"im": {
			"chat-search":           {args: []string{"im", "+chat-search"}, queryArg: "--keyword"},
			"messages-search":       {args: []string{"im", "+messages-search"}, queryArg: "--keyword"},
			"chat-messages-list":    {args: []string{"im", "+chat-messages-list"}},
			"threads-messages-list": {args: []string{"im", "+threads-messages-list"}},
		},
		"mail": {
			"triage":  {args: []string{"mail", "+triage"}, queryArg: "--query"},
			"thread":  {args: []string{"mail", "+thread"}},
			"message": {args: []string{"mail", "+message"}},
		},
		"minutes": {
			"get":      {args: []string{"minutes", "minutes", "get"}},
			"download": {args: []string{"minutes", "+download"}},
		},
		"vc": {
			"search":    {args: []string{"vc", "+search"}},
			"notes":     {args: []string{"vc", "+notes"}},
			"recording": {args: []string{"vc", "+recording"}},
		},
		"meetings": {
			"search":    {args: []string{"vc", "+search"}},
			"notes":     {args: []string{"vc", "+notes"}},
			"recording": {args: []string{"vc", "+recording"}},
		},
		"calendar": {
			"agenda":     {args: []string{"calendar", "+agenda"}},
			"freebusy":   {args: []string{"calendar", "+freebusy"}},
			"suggestion": {args: []string{"calendar", "+suggestion"}},
		},
		"tasks": {
			"get-my-tasks": {args: []string{"task", "+get-my-tasks"}},
		},
		"wiki": {
			"spaces": {args: []string{"wiki", "spaces", "list"}, pageAll: true},
			"nodes":  {args: []string{"wiki", "nodes", "list"}, pageAll: true},
		},
		"sheets": {
			"info": {args: []string{"sheets", "+info"}},
			"read": {args: []string{"sheets", "+read"}},
			"find": {args: []string{"sheets", "+find"}},
		},
		"base": {
			"data-query":     {args: []string{"base", "+data-query"}},
			"table-list":     {args: []string{"base", "+table-list"}},
			"record-list":    {args: []string{"base", "+record-list"}},
			"field-list":     {args: []string{"base", "+field-list"}},
			"view-list":      {args: []string{"base", "+view-list"}},
			"dashboard-list": {args: []string{"base", "+dashboard-list"}},
			"workflow-list":  {args: []string{"base", "+workflow-list"}},
		},
		"_system": {
			"schema": {args: []string{"schema"}},
			"doctor": {args: []string{"doctor", "--format", "json"}},
		},
	}
	spec, ok := specs[domain][action]
	return spec, ok
}

func actionSpecFor(domain, action string) (actionSpec, bool) {
	if domain == "_system" {
		switch action {
		case "api":
			return actionSpec{args: []string{"api"}}, true
		case "schema":
			return actionSpec{args: []string{"schema"}}, true
		case "doctor":
			return actionSpec{args: []string{"doctor"}}, true
		case "auth":
			return actionSpec{args: []string{"auth"}}, true
		case "config":
			return actionSpec{args: []string{"config"}}, true
		case "profile":
			return actionSpec{args: []string{"profile"}}, true
		case "event-subscribe":
			return actionSpec{args: []string{"event", "+subscribe"}}, true
		}
	}
	spec, ok := domainActionSpecs()[domain][action]
	return spec, ok
}

func domainActionSpecs() map[string]map[string]actionSpec {
	return map[string]map[string]actionSpec{
		"drive": {
			"upload":      {args: []string{"drive", "+upload"}},
			"download":    {args: []string{"drive", "+download"}},
			"import":      {args: []string{"drive", "+import"}},
			"export":      {args: []string{"drive", "+export"}},
			"move":        {args: []string{"drive", "+move"}},
			"delete":      {args: []string{"drive", "files", "delete"}},
			"add-comment": {args: []string{"drive", "+add-comment"}},
			"task-result": {args: []string{"drive", "+task_result"}},
		},
		"wiki": {"node-create": {args: []string{"wiki", "+node-create"}}},
		"im": {
			"chat-create":                 {args: []string{"im", "+chat-create"}},
			"chat-update":                 {args: []string{"im", "+chat-update"}},
			"messages-send":               {args: []string{"im", "+messages-send"}},
			"messages-reply":              {args: []string{"im", "+messages-reply"}},
			"messages-resources-download": {args: []string{"im", "+messages-resources-download"}},
			"reactions":                   {args: []string{"im", "reactions"}},
			"pins":                        {args: []string{"im", "pins"}},
		},
		"calendar": {
			"create": {args: []string{"calendar", "+create"}},
			"rsvp":   {args: []string{"calendar", "+rsvp"}},
		},
		"tasks": {
			"create":            {args: []string{"task", "+create"}},
			"update":            {args: []string{"task", "+update"}},
			"assign":            {args: []string{"task", "+assign"}},
			"comment":           {args: []string{"task", "+comment"}},
			"complete":          {args: []string{"task", "+complete"}},
			"reopen":            {args: []string{"task", "+reopen"}},
			"followers":         {args: []string{"task", "+followers"}},
			"reminder":          {args: []string{"task", "+reminder"}},
			"tasklist-create":   {args: []string{"task", "+tasklist-create"}},
			"tasklist-task-add": {args: []string{"task", "+tasklist-task-add"}},
			"subtask":           {args: []string{"task", "subtasks"}},
		},
		"mail": {
			"send":         {args: []string{"mail", "+send"}},
			"draft-create": {args: []string{"mail", "+draft-create"}},
			"draft-edit":   {args: []string{"mail", "+draft-edit"}},
			"reply":        {args: []string{"mail", "+reply"}},
			"reply-all":    {args: []string{"mail", "+reply-all"}},
			"forward":      {args: []string{"mail", "+forward"}},
			"watch":        {args: []string{"mail", "+watch"}},
		},
		"approval": {
			"approve":  {args: []string{"approval", "tasks", "approve"}},
			"reject":   {args: []string{"approval", "tasks", "reject"}},
			"transfer": {args: []string{"approval", "tasks", "transfer"}},
			"comment":  {args: []string{"approval", "tasks", "comment"}},
		},
		"base": {
			"base-create":      {args: []string{"base", "+base-create"}},
			"base-copy":        {args: []string{"base", "+base-copy"}},
			"table-create":     {args: []string{"base", "+table-create"}},
			"table-update":     {args: []string{"base", "+table-update"}},
			"table-delete":     {args: []string{"base", "+table-delete"}},
			"record-upsert":    {args: []string{"base", "+record-upsert"}},
			"record-delete":    {args: []string{"base", "+record-delete"}},
			"field-create":     {args: []string{"base", "+field-create"}},
			"field-update":     {args: []string{"base", "+field-update"}},
			"field-delete":     {args: []string{"base", "+field-delete"}},
			"view-create":      {args: []string{"base", "+view-create"}},
			"view-delete":      {args: []string{"base", "+view-delete"}},
			"dashboard-create": {args: []string{"base", "+dashboard-create"}},
			"workflow-create":  {args: []string{"base", "+workflow-create"}},
		},
		"docs": {
			"create":            {args: []string{"docs", "+create"}},
			"update":            {args: []string{"docs", "+update"}},
			"media-download":    {args: []string{"docs", "+media-download"}},
			"media-insert":      {args: []string{"docs", "+media-insert"}},
			"media-preview":     {args: []string{"docs", "+media-preview"}},
			"whiteboard-update": {args: []string{"docs", "+whiteboard-update"}},
		},
		"minutes": {"download": {args: []string{"minutes", "+download"}}},
		"vc": {
			"notes":     {args: []string{"vc", "+notes"}},
			"recording": {args: []string{"vc", "+recording"}},
		},
		"sheets": {
			"create":      {args: []string{"sheets", "+create"}},
			"append":      {args: []string{"sheets", "+append"}},
			"write":       {args: []string{"sheets", "+write"}},
			"write-image": {args: []string{"sheets", "+write-image"}},
			"export":      {args: []string{"sheets", "+export"}},
		},
		"contact": {
			"search-user": {args: []string{"contact", "+search-user"}},
			"get-user":    {args: []string{"contact", "+get-user"}},
		},
	}
}

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
		return nil, fmt.Errorf("not found: %s", nodePath)
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

var _ clipkg.Runner = (*clipkg.Executor)(nil)
