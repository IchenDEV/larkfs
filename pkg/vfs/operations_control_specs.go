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
	case "attendance":
		return []string{"user-tasks"}
	case "base":
		return []string{"data-query", "base-get", "base-block-list", "table-list", "table-get", "record-list", "record-get", "record-search", "field-list", "field-get", "view-list", "dashboard-list", "workflow-list", "form-list", "role-list"}
	case "drive":
		return []string{"search", "inspect", "comments", "statistics", "view-records", "metas", "cover", "preview", "secure-label-list", "status", "version-history"}
	case "event":
		return []string{"list", "schema", "status"}
	case "im":
		return []string{"chat-search", "messages-search", "chat-messages-list", "threads-messages-list"}
	case "mail":
		return []string{"triage", "thread", "message", "messages", "signature", "lint-html"}
	case "markdown":
		return []string{"fetch", "diff"}
	case "minutes":
		return []string{"get", "search", "download"}
	case "okr":
		return []string{"cycle-list", "cycle-detail", "progress-get", "progress-list"}
	case "vc", "meetings":
		return []string{"search", "notes", "recording", "meeting-events"}
	case "calendar":
		return []string{"agenda", "freebusy", "suggestion"}
	case "tasks":
		return []string{"get-my-tasks"}
	case "wiki":
		return []string{"spaces", "nodes"}
	case "sheets":
		return []string{"workbook-info", "csv-get", "cells-get", "cells-search", "sheet-info", "chart-list", "filter-list", "filter-view-list", "float-image-list", "pivot-list", "sparkline-list"}
	case "whiteboard":
		return []string{"query"}
	case "_system":
		return []string{"schema", "doctor", "event-list", "event-status"}
	default:
		return nil
	}
}

func opActions(domain string) []string {
	switch domain {
	case "drive":
		return []string{"upload", "download", "import", "export", "export-download", "move", "delete", "replace", "add-comment", "apply-permission", "create-folder", "create-shortcut", "pull", "push", "secure-label-update", "sync", "task-result", "version-delete", "version-get", "version-revert"}
	case "wiki":
		return []string{"node-create"}
	case "im":
		return []string{"chat-create", "chat-update", "messages-send", "messages-reply", "reactions", "pins", "messages-resources-download"}
	case "calendar":
		return []string{"create", "rsvp"}
	case "tasks":
		return []string{"create", "update", "assign", "comment", "complete", "reopen", "followers", "reminder", "tasklist-create", "tasklist-task-add", "subtask"}
	case "mail":
		return []string{"send", "draft-create", "draft-edit", "draft-send", "reply", "reply-all", "forward", "send-receipt", "decline-receipt", "share-to-chat", "template-create", "template-update", "watch"}
	case "approval":
		return []string{"approve", "reject", "transfer", "comment"}
	case "base":
		return []string{"advperm-enable", "advperm-disable", "base-create", "base-copy", "base-block-create", "base-block-delete", "base-block-move", "base-block-rename", "table-create", "table-update", "table-delete", "record-upsert", "record-delete", "record-batch-create", "record-batch-update", "record-upload-attachment", "record-remove-attachment", "field-create", "field-update", "field-delete", "view-create", "view-delete", "dashboard-create", "dashboard-update", "dashboard-delete", "dashboard-arrange", "workflow-create", "workflow-update", "workflow-enable", "workflow-disable", "form-create", "form-update", "form-delete", "form-submit", "role-create", "role-update", "role-delete"}
	case "docs":
		return []string{"create", "update", "media-download", "media-insert", "media-preview", "media-upload", "whiteboard-update"}
	case "event":
		return []string{"consume", "stop"}
	case "markdown":
		return []string{"create", "overwrite", "patch"}
	case "minutes":
		return []string{"download", "speaker-replace", "update", "upload"}
	case "okr":
		return []string{"progress-create", "progress-update", "progress-delete", "upload-image"}
	case "vc", "meetings":
		return []string{"meeting-join", "meeting-leave", "notes", "recording"}
	case "sheets":
		return []string{"workbook-create", "workbook-export", "csv-put", "cells-set", "cells-batch-clear", "cells-batch-set-style", "cells-clear", "cells-merge", "cells-replace", "cells-set-image", "cells-set-style", "cells-unmerge", "cols-resize", "rows-resize", "dim-delete", "dim-freeze", "dim-group", "dim-hide", "dim-insert", "dim-move", "dim-ungroup", "dim-unhide", "dropdown-delete", "dropdown-set", "dropdown-update", "filter-create", "filter-delete", "filter-update", "filter-view-create", "filter-view-delete", "filter-view-update", "float-image-create", "float-image-delete", "float-image-update", "pivot-create", "pivot-delete", "pivot-update", "range-copy", "range-fill", "range-move", "range-sort", "sheet-copy", "sheet-create", "sheet-delete", "sheet-hide", "sheet-move", "sheet-rename", "sheet-set-tab-color", "sheet-unhide", "sparkline-create", "sparkline-delete", "sparkline-update"}
	case "slides":
		return []string{"create", "media-upload", "replace-slide"}
	case "whiteboard":
		return []string{"update"}
	case "contact":
		return []string{"search-user", "get-user"}
	case "_system":
		return []string{"api", "schema", "doctor", "auth", "config", "profile", "event-consume", "event-stop", "update"}
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
		"attendance": {
			"user-tasks": {args: []string{"attendance", "user_tasks", "query"}, pageAll: true},
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
			"search":            {args: []string{"drive", "+search"}, queryArg: "--query"},
			"inspect":           {args: []string{"drive", "+inspect"}},
			"comments":          {args: []string{"drive", "file.comments", "list"}, pageAll: true},
			"statistics":        {args: []string{"drive", "file.statistics", "get"}},
			"view-records":      {args: []string{"drive", "file.view_records", "list"}, pageAll: true},
			"metas":             {args: []string{"drive", "metas", "batch_query"}},
			"cover":             {args: []string{"drive", "+cover"}},
			"preview":           {args: []string{"drive", "+preview"}},
			"secure-label-list": {args: []string{"drive", "+secure-label-list"}},
			"status":            {args: []string{"drive", "+status"}},
			"version-history":   {args: []string{"drive", "+version-history"}},
		},
		"event": {
			"list":   {args: []string{"event", "list", "--json"}},
			"schema": {args: []string{"event", "schema"}},
			"status": {args: []string{"event", "status", "--json"}},
		},
		"im": {
			"chat-search":           {args: []string{"im", "+chat-search"}, queryArg: "--keyword"},
			"messages-search":       {args: []string{"im", "+messages-search"}, queryArg: "--keyword"},
			"chat-messages-list":    {args: []string{"im", "+chat-messages-list"}},
			"threads-messages-list": {args: []string{"im", "+threads-messages-list"}},
		},
		"mail": {
			"triage":    {args: []string{"mail", "+triage"}, queryArg: "--query"},
			"thread":    {args: []string{"mail", "+thread"}},
			"message":   {args: []string{"mail", "+message"}},
			"messages":  {args: []string{"mail", "+messages"}},
			"signature": {args: []string{"mail", "+signature"}},
			"lint-html": {args: []string{"mail", "+lint-html"}},
		},
		"markdown": {
			"fetch": {args: []string{"markdown", "+fetch"}},
			"diff":  {args: []string{"markdown", "+diff"}},
		},
		"minutes": {
			"get":      {args: []string{"minutes", "minutes", "get"}},
			"search":   {args: []string{"minutes", "+search"}, queryArg: "--query"},
			"download": {args: []string{"minutes", "+download"}},
		},
		"okr": {
			"cycle-list":    {args: []string{"okr", "+cycle-list"}},
			"cycle-detail":  {args: []string{"okr", "+cycle-detail"}},
			"progress-get":  {args: []string{"okr", "+progress-get"}},
			"progress-list": {args: []string{"okr", "+progress-list"}},
		},
		"vc": {
			"search":         {args: []string{"vc", "+search"}},
			"notes":          {args: []string{"vc", "+notes"}},
			"recording":      {args: []string{"vc", "+recording"}},
			"meeting-events": {args: []string{"vc", "+meeting-events"}},
		},
		"meetings": {
			"search":         {args: []string{"vc", "+search"}},
			"notes":          {args: []string{"vc", "+notes"}},
			"recording":      {args: []string{"vc", "+recording"}},
			"meeting-events": {args: []string{"vc", "+meeting-events"}},
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
			"workbook-info":    {args: []string{"sheets", "+workbook-info"}},
			"csv-get":          {args: []string{"sheets", "+csv-get"}},
			"cells-get":        {args: []string{"sheets", "+cells-get"}},
			"cells-search":     {args: []string{"sheets", "+cells-search"}},
			"sheet-info":       {args: []string{"sheets", "+sheet-info"}},
			"chart-list":       {args: []string{"sheets", "+chart-list"}},
			"filter-list":      {args: []string{"sheets", "+filter-list"}},
			"filter-view-list": {args: []string{"sheets", "+filter-view-list"}},
			"float-image-list": {args: []string{"sheets", "+float-image-list"}},
			"pivot-list":       {args: []string{"sheets", "+pivot-list"}},
			"sparkline-list":   {args: []string{"sheets", "+sparkline-list"}},
		},
		"base": {
			"data-query":      {args: []string{"base", "+data-query"}},
			"base-get":        {args: []string{"base", "+base-get"}},
			"base-block-list": {args: []string{"base", "+base-block-list"}},
			"table-list":      {args: []string{"base", "+table-list"}},
			"table-get":       {args: []string{"base", "+table-get"}},
			"record-list":     {args: []string{"base", "+record-list"}},
			"record-get":      {args: []string{"base", "+record-get"}},
			"record-search":   {args: []string{"base", "+record-search"}},
			"field-list":      {args: []string{"base", "+field-list"}},
			"field-get":       {args: []string{"base", "+field-get"}},
			"view-list":       {args: []string{"base", "+view-list"}},
			"dashboard-list":  {args: []string{"base", "+dashboard-list"}},
			"workflow-list":   {args: []string{"base", "+workflow-list"}},
			"form-list":       {args: []string{"base", "+form-list"}},
			"role-list":       {args: []string{"base", "+role-list"}},
		},
		"_system": {
			"schema":       {args: []string{"schema"}},
			"doctor":       {args: []string{"doctor", "--format", "json"}},
			"event-list":   {args: []string{"event", "list", "--json"}},
			"event-status": {args: []string{"event", "status", "--json"}},
		},
		"whiteboard": {
			"query": {args: []string{"whiteboard", "+query"}},
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
		case "event-consume":
			return actionSpec{args: []string{"event", "consume"}}, true
		case "event-stop":
			return actionSpec{args: []string{"event", "stop"}}, true
		case "update":
			return actionSpec{args: []string{"update"}}, true
		}
	}
	spec, ok := domainActionSpecs()[domain][action]
	return spec, ok
}

var _ clipkg.Runner = (*clipkg.Executor)(nil)
