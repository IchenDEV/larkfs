package vfs

func querySpec(domain, action string) (actionSpec, bool) {
	specs := map[string]map[string]actionSpec{
		"apps":       appsQuerySpecs(),
		"approval":   approvalQuerySpecs(),
		"attendance": {"user-tasks": {args: []string{"attendance", "user_tasks", "query"}, pageAll: true}},
		"base":       baseQuerySpecs(),
		"calendar": {
			"agenda":     {args: []string{"calendar", "+agenda"}},
			"freebusy":   {args: []string{"calendar", "+freebusy"}},
			"room-find":  {args: []string{"calendar", "+room-find"}},
			"suggestion": {args: []string{"calendar", "+suggestion"}},
		},
		"contact": {
			"search-user": {args: []string{"contact", "+search-user"}, queryArg: "--query"},
			"get-user":    {args: []string{"contact", "+get-user"}},
		},
		"docs": {
			"search": {args: []string{"docs", "+search"}, queryArg: "--query"},
			"fetch":  {args: []string{"docs", "+fetch", "--api-version", "v2"}},
		},
		"drive":    driveQuerySpecs(),
		"event":    eventQuerySpecs(),
		"im":       imQuerySpecs(),
		"mail":     mailQuerySpecs(),
		"markdown": {"fetch": {args: []string{"markdown", "+fetch"}}, "diff": {args: []string{"markdown", "+diff"}}},
		"meetings": vcQuerySpecs(),
		"minutes": {
			"get":      {args: []string{"minutes", "minutes", "get"}},
			"search":   {args: []string{"minutes", "+search"}, queryArg: "--query"},
			"download": {args: []string{"minutes", "+download"}},
		},
		"note": {
			"detail":     {args: []string{"note", "+detail"}},
			"transcript": {args: []string{"note", "+transcript"}},
		},
		"okr": {
			"cycle-list":    {args: []string{"okr", "+cycle-list"}},
			"cycle-detail":  {args: []string{"okr", "+cycle-detail"}},
			"progress-get":  {args: []string{"okr", "+progress-get"}},
			"progress-list": {args: []string{"okr", "+progress-list"}},
		},
		"sheets":     sheetsQuerySpecs(),
		"tasks":      taskQuerySpecs(),
		"vc":         vcQuerySpecs(),
		"whiteboard": {"query": {args: []string{"whiteboard", "+query"}}},
		"wiki":       wikiQuerySpecs(),
		"_system":    systemQuerySpecs(),
	}
	spec, ok := specs[domain][action]
	return spec, ok
}

func approvalQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"instances": {args: []string{"approval", "instances", "list"}, pageAll: true},
		"tasks":     {args: []string{"approval", "tasks", "list"}, pageAll: true},
	}
}

func appsQuerySpecs() map[string]actionSpec {
	return plusActionSpecs("apps", appsQueryActionNames())
}

func driveQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
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
	}
}

func eventQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"list":   {args: []string{"event", "list", "--json"}},
		"schema": {args: []string{"event", "schema"}, queryPos: true},
		"status": {args: []string{"event", "status", "--json"}},
	}
}

func systemQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"schema":       {args: []string{"schema"}, queryPos: true},
		"doctor":       {args: []string{"doctor", "--format", "json"}},
		"event-list":   {args: []string{"event", "list", "--json"}},
		"event-schema": {args: []string{"event", "schema"}, queryPos: true},
		"event-status": {args: []string{"event", "status", "--json"}},
		"skills-list":  {args: []string{"skills", "list", "--json"}, queryPos: true},
		"skills-read":  {args: []string{"skills", "read", "--json"}, queryPos: true},
	}
}

func imQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"chat-list":             {args: []string{"im", "+chat-list"}},
		"chat-search":           {args: []string{"im", "+chat-search"}, queryArg: "--keyword"},
		"messages-search":       {args: []string{"im", "+messages-search"}, queryArg: "--keyword"},
		"messages-mget":         {args: []string{"im", "+messages-mget"}},
		"chat-messages-list":    {args: []string{"im", "+chat-messages-list"}},
		"threads-messages-list": {args: []string{"im", "+threads-messages-list"}},
		"feed-group-list":       {args: []string{"im", "+feed-group-list"}},
		"feed-group-list-item":  {args: []string{"im", "+feed-group-list-item"}},
		"feed-group-query-item": {args: []string{"im", "+feed-group-query-item"}},
		"feed-shortcut-list":    {args: []string{"im", "+feed-shortcut-list"}},
		"flag-list":             {args: []string{"im", "+flag-list"}},
	}
}

func mailQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"triage":    {args: []string{"mail", "+triage"}, queryArg: "--query"},
		"thread":    {args: []string{"mail", "+thread"}},
		"message":   {args: []string{"mail", "+message"}},
		"messages":  {args: []string{"mail", "+messages"}},
		"signature": {args: []string{"mail", "+signature"}},
		"lint-html": {args: []string{"mail", "+lint-html"}},
	}
}

func vcQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"search":              {args: []string{"vc", "+search"}},
		"notes":               {args: []string{"vc", "+notes"}},
		"recording":           {args: []string{"vc", "+recording"}},
		"meeting-events":      {args: []string{"vc", "+meeting-events"}},
		"meeting-list-active": {args: []string{"vc", "+meeting-list-active"}},
	}
}

func taskQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"get-my-tasks":      {args: []string{"task", "+get-my-tasks"}},
		"get-related-tasks": {args: []string{"task", "+get-related-tasks"}},
		"search":            {args: []string{"task", "+search"}, queryArg: "--query"},
		"tasklist-search":   {args: []string{"task", "+tasklist-search"}, queryArg: "--query"},
	}
}

func wikiQuerySpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"spaces":      {args: []string{"wiki", "spaces", "list"}, pageAll: true},
		"nodes":       {args: []string{"wiki", "nodes", "list"}, pageAll: true},
		"space-list":  {args: []string{"wiki", "+space-list"}},
		"node-list":   {args: []string{"wiki", "+node-list"}},
		"node-get":    {args: []string{"wiki", "+node-get"}},
		"member-list": {args: []string{"wiki", "+member-list"}},
	}
}
