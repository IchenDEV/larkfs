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
	queryPos bool
	pageAll  bool
}

func plusActionSpecs(domain string, actions []string) map[string]actionSpec {
	specs := make(map[string]actionSpec, len(actions))
	for _, action := range actions {
		specs[action] = actionSpec{args: []string{domain, "+" + action}}
	}
	return specs
}

func appsQueryActionNames() []string {
	return []string{
		"list",
		"db-table-list",
		"db-table-get",
		"release-get",
		"release-list",
		"access-scope-get",
		"session-get",
		"session-messages-list",
	}
}

func appsOpActionNames() []string {
	return []string{
		"create",
		"update",
		"html-publish",
		"init",
		"git-credential-init",
		"env-pull",
		"db-execute",
		"db-env-create",
		"release-create",
		"access-scope-set",
		"session-create",
		"chat",
	}
}

func queryActions(domain string) []string {
	switch domain {
	case "apps":
		return appsQueryActionNames()
	case "contact":
		return []string{"search-user", "get-user"}
	case "docs":
		return []string{"search", "fetch"}
	case "approval":
		return []string{"instances", "tasks"}
	case "attendance":
		return []string{"user-tasks"}
	case "base":
		return baseQueryActionNames()
	case "drive":
		return []string{"search", "inspect", "comments", "statistics", "view-records", "metas", "cover", "preview", "secure-label-list", "status", "version-history"}
	case "event":
		return []string{"list", "schema", "status"}
	case "im":
		return []string{"chat-list", "chat-search", "messages-search", "messages-mget", "chat-messages-list", "threads-messages-list", "feed-group-list", "feed-group-list-item", "feed-group-query-item", "feed-shortcut-list", "flag-list"}
	case "mail":
		return []string{"triage", "thread", "message", "messages", "signature", "lint-html"}
	case "markdown":
		return []string{"fetch", "diff"}
	case "minutes":
		return []string{"get", "search", "download"}
	case "note":
		return []string{"detail", "transcript"}
	case "okr":
		return []string{"cycle-list", "cycle-detail", "progress-get", "progress-list"}
	case "vc", "meetings":
		return []string{"search", "notes", "recording", "meeting-events", "meeting-list-active"}
	case "calendar":
		return []string{"agenda", "freebusy", "room-find", "suggestion"}
	case "tasks":
		return []string{"get-my-tasks", "get-related-tasks", "search", "tasklist-search"}
	case "wiki":
		return []string{"spaces", "nodes", "space-list", "node-list", "node-get", "member-list"}
	case "sheets":
		return sheetsQueryActionNames()
	case "whiteboard":
		return []string{"query"}
	case "_system":
		return []string{"schema", "doctor", "event-list", "event-schema", "event-status", "skills-list", "skills-read"}
	default:
		return nil
	}
}

func opActions(domain string) []string {
	switch domain {
	case "apps":
		return appsOpActionNames()
	case "drive":
		return []string{"upload", "download", "import", "export", "export-download", "move", "delete", "replace", "add-comment", "apply-permission", "member-add", "create-folder", "create-shortcut", "pull", "push", "secure-label-update", "sync", "task-result", "version-delete", "version-get", "version-revert"}
	case "wiki":
		return []string{"space-create", "delete-space", "member-add", "member-remove", "move", "node-copy", "node-create", "node-delete"}
	case "im":
		return []string{"chat-create", "chat-update", "messages-send", "messages-reply", "reactions", "pins", "messages-resources-download", "feed-shortcut-create", "feed-shortcut-remove", "flag-create", "flag-cancel"}
	case "calendar":
		return []string{"create", "update", "rsvp"}
	case "tasks":
		return []string{"create", "update", "assign", "comment", "complete", "reopen", "followers", "reminder", "set-ancestor", "subscribe-event", "tasklist-create", "tasklist-members", "tasklist-task-add", "subtask", "upload-attachment"}
	case "mail":
		return []string{"send", "draft-create", "draft-edit", "draft-send", "reply", "reply-all", "forward", "send-receipt", "decline-receipt", "share-to-chat", "template-create", "template-update", "watch"}
	case "approval":
		return []string{"approve", "reject", "transfer", "comment"}
	case "base":
		return baseOpActionNames()
	case "docs":
		return []string{"create", "update", "media-download", "media-insert", "media-preview", "media-upload", "resource-delete", "resource-download", "resource-update", "whiteboard-update"}
	case "event":
		return []string{"consume", "stop"}
	case "markdown":
		return []string{"create", "overwrite", "patch"}
	case "minutes":
		return []string{"download", "speaker-replace", "summary", "todo", "update", "upload", "word-replace"}
	case "okr":
		return []string{"batch-create", "indicator-update", "progress-create", "progress-update", "progress-delete", "reorder", "upload-image", "weight"}
	case "vc", "meetings":
		return []string{"meeting-join", "meeting-leave", "notes", "recording"}
	case "sheets":
		return sheetsOpActionNames()
	case "slides":
		return []string{"create", "media-upload", "replace-slide", "screenshot"}
	case "whiteboard":
		return []string{"update"}
	case "contact":
		return []string{"search-user", "get-user"}
	case "_system":
		return []string{"api", "schema", "doctor", "auth", "config", "profile", "event-consume", "event-stop", "skills-read", "update"}
	default:
		return nil
	}
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
		case "skills-read":
			return actionSpec{args: []string{"skills", "read", "--json"}, queryPos: true}, true
		case "update":
			return actionSpec{args: []string{"update"}}, true
		}
	}
	spec, ok := domainActionSpecs()[domain][action]
	return spec, ok
}

var _ clipkg.Runner = (*clipkg.Executor)(nil)
