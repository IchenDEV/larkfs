package vfs

func domainActionSpecs() map[string]map[string]actionSpec {
	return map[string]map[string]actionSpec{
		"apps": appsActionSpecs(),
		"approval": {
			"approve":  {args: []string{"approval", "tasks", "approve"}},
			"reject":   {args: []string{"approval", "tasks", "reject"}},
			"transfer": {args: []string{"approval", "tasks", "transfer"}},
			"comment":  {args: []string{"approval", "tasks", "comment"}},
		},
		"base": baseActionSpecs(),
		"calendar": {
			"create": {args: []string{"calendar", "+create"}},
			"update": {args: []string{"calendar", "+update"}},
			"rsvp":   {args: []string{"calendar", "+rsvp"}},
		},
		"contact": {
			"search-user": {args: []string{"contact", "+search-user"}},
			"get-user":    {args: []string{"contact", "+get-user"}},
		},
		"docs":  docsActionSpecs(),
		"drive": driveActionSpecs(),
		"event": {
			"consume": {args: []string{"event", "consume"}},
			"stop":    {args: []string{"event", "stop"}},
		},
		"im": {
			"chat-create":                 {args: []string{"im", "+chat-create"}},
			"chat-update":                 {args: []string{"im", "+chat-update"}},
			"messages-send":               {args: []string{"im", "+messages-send"}},
			"messages-reply":              {args: []string{"im", "+messages-reply"}},
			"messages-resources-download": {args: []string{"im", "+messages-resources-download"}},
			"feed-shortcut-create":        {args: []string{"im", "+feed-shortcut-create"}},
			"feed-shortcut-remove":        {args: []string{"im", "+feed-shortcut-remove"}},
			"flag-create":                 {args: []string{"im", "+flag-create"}},
			"flag-cancel":                 {args: []string{"im", "+flag-cancel"}},
			"reactions":                   {args: []string{"im", "reactions"}},
			"pins":                        {args: []string{"im", "pins"}},
		},
		"mail": {
			"send":            {args: []string{"mail", "+send"}},
			"draft-create":    {args: []string{"mail", "+draft-create"}},
			"draft-edit":      {args: []string{"mail", "+draft-edit"}},
			"draft-send":      {args: []string{"mail", "+draft-send"}},
			"reply":           {args: []string{"mail", "+reply"}},
			"reply-all":       {args: []string{"mail", "+reply-all"}},
			"forward":         {args: []string{"mail", "+forward"}},
			"send-receipt":    {args: []string{"mail", "+send-receipt"}},
			"decline-receipt": {args: []string{"mail", "+decline-receipt"}},
			"share-to-chat":   {args: []string{"mail", "+share-to-chat"}},
			"template-create": {args: []string{"mail", "+template-create"}},
			"template-update": {args: []string{"mail", "+template-update"}},
			"watch":           {args: []string{"mail", "+watch"}},
		},
		"markdown": {
			"create":    {args: []string{"markdown", "+create"}},
			"overwrite": {args: []string{"markdown", "+overwrite"}},
			"patch":     {args: []string{"markdown", "+patch"}},
		},
		"meetings": vcActionSpecs(),
		"minutes": {
			"download":        {args: []string{"minutes", "+download"}},
			"speaker-replace": {args: []string{"minutes", "+speaker-replace"}},
			"summary":         {args: []string{"minutes", "+summary"}},
			"todo":            {args: []string{"minutes", "+todo"}},
			"update":          {args: []string{"minutes", "+update"}},
			"upload":          {args: []string{"minutes", "+upload"}},
			"word-replace":    {args: []string{"minutes", "+word-replace"}},
		},
		"okr":        okrActionSpecs(),
		"sheets":     sheetsActionSpecs(),
		"slides":     slidesActionSpecs(),
		"tasks":      taskActionSpecs(),
		"vc":         vcActionSpecs(),
		"whiteboard": {"update": {args: []string{"whiteboard", "+update"}}},
		"wiki":       wikiActionSpecs(),
	}
}

func appsActionSpecs() map[string]actionSpec {
	return plusActionSpecs("apps", appsOpActionNames())
}

func docsActionSpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"create":            {args: []string{"docs", "+create", "--api-version", "v2"}},
		"update":            {args: []string{"docs", "+update", "--api-version", "v2"}},
		"media-download":    {args: []string{"docs", "+media-download"}},
		"media-insert":      {args: []string{"docs", "+media-insert"}},
		"media-preview":     {args: []string{"docs", "+media-preview"}},
		"media-upload":      {args: []string{"docs", "+media-upload"}},
		"resource-delete":   {args: []string{"docs", "+resource-delete"}},
		"resource-download": {args: []string{"docs", "+resource-download"}},
		"resource-update":   {args: []string{"docs", "+resource-update"}},
		"whiteboard-update": {args: []string{"docs", "+whiteboard-update"}},
	}
}

func driveActionSpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"upload":              {args: []string{"drive", "+upload"}},
		"download":            {args: []string{"drive", "+download"}},
		"import":              {args: []string{"drive", "+import"}},
		"export":              {args: []string{"drive", "+export"}},
		"export-download":     {args: []string{"drive", "+export-download"}},
		"move":                {args: []string{"drive", "+move"}},
		"delete":              {args: []string{"drive", "+delete"}},
		"add-comment":         {args: []string{"drive", "+add-comment"}},
		"apply-permission":    {args: []string{"drive", "+apply-permission"}},
		"member-add":          {args: []string{"drive", "+member-add"}},
		"create-folder":       {args: []string{"drive", "+create-folder"}},
		"create-shortcut":     {args: []string{"drive", "+create-shortcut"}},
		"pull":                {args: []string{"drive", "+pull"}},
		"push":                {args: []string{"drive", "+push"}},
		"secure-label-update": {args: []string{"drive", "+secure-label-update"}},
		"sync":                {args: []string{"drive", "+sync"}},
		"task_result":         {args: []string{"drive", "+task_result"}},
		"task-result":         {args: []string{"drive", "+task_result"}},
		"version-delete":      {args: []string{"drive", "+version-delete"}},
		"version-get":         {args: []string{"drive", "+version-get"}},
		"version-revert":      {args: []string{"drive", "+version-revert"}},
	}
}

func okrActionSpecs() map[string]actionSpec {
	return plusActionSpecs("okr", []string{
		"batch-create",
		"indicator-update",
		"progress-create",
		"progress-update",
		"progress-delete",
		"reorder",
		"upload-image",
		"weight",
	})
}

func slidesActionSpecs() map[string]actionSpec {
	return plusActionSpecs("slides", slidesOpActionNames())
}

func taskActionSpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"create":            {args: []string{"task", "+create"}},
		"update":            {args: []string{"task", "+update"}},
		"assign":            {args: []string{"task", "+assign"}},
		"comment":           {args: []string{"task", "+comment"}},
		"complete":          {args: []string{"task", "+complete"}},
		"reopen":            {args: []string{"task", "+reopen"}},
		"followers":         {args: []string{"task", "+followers"}},
		"reminder":          {args: []string{"task", "+reminder"}},
		"set-ancestor":      {args: []string{"task", "+set-ancestor"}},
		"subscribe-event":   {args: []string{"task", "+subscribe-event"}},
		"tasklist-create":   {args: []string{"task", "+tasklist-create"}},
		"tasklist-members":  {args: []string{"task", "+tasklist-members"}},
		"tasklist-task-add": {args: []string{"task", "+tasklist-task-add"}},
		"upload-attachment": {args: []string{"task", "+upload-attachment"}},
		"subtask":           {args: []string{"task", "subtasks"}},
	}
}

func vcActionSpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"meeting-join":  {args: []string{"vc", "+meeting-join"}},
		"meeting-leave": {args: []string{"vc", "+meeting-leave"}},
		"notes":         {args: []string{"vc", "+notes"}},
		"recording":     {args: []string{"vc", "+recording"}},
	}
}

func wikiActionSpecs() map[string]actionSpec {
	return map[string]actionSpec{
		"space-create":  {args: []string{"wiki", "+space-create"}},
		"delete-space":  {args: []string{"wiki", "+delete-space"}},
		"member-add":    {args: []string{"wiki", "+member-add"}},
		"member-remove": {args: []string{"wiki", "+member-remove"}},
		"move":          {args: []string{"wiki", "+move"}},
		"node-copy":     {args: []string{"wiki", "+node-copy"}},
		"node-create":   {args: []string{"wiki", "+node-create"}},
		"node-delete":   {args: []string{"wiki", "+node-delete"}},
	}
}
