package vfs

import clipkg "github.com/IchenDEV/larkfs/pkg/cli"

type execRequest struct {
	Args       []string       `json:"args,omitempty"`
	Flags      map[string]any `json:"flags,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	Query      string         `json:"query,omitempty"`
	TargetPath string         `json:"target_path,omitempty"`
}

type actionSpec struct {
	args     []string
	queryArg string
	pageAll  bool
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
		return []string{"upload", "download", "import", "export", "move", "delete", "replace", "add-comment", "task-result"}
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

var _ clipkg.Runner = (*clipkg.Executor)(nil)
