package vfs

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/adapter"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Operations struct {
	cli        clipkg.Runner
	tree       *Tree
	drive      *adapter.DriveAdapter
	wiki       *adapter.WikiAdapter
	calendar   *adapter.CalendarAdapter
	task       *adapter.TaskAdapter
	im         *adapter.IMAdapter
	mail       *adapter.MailAdapter
	meeting    *adapter.MeetingAdapter
	controls   *controlStore
	readOnly   bool
	refreshTTL time.Duration
}

type OperationsConfig struct {
	CLI      clipkg.Runner
	Tree     *Tree
	Drive    *adapter.DriveAdapter
	Wiki     *adapter.WikiAdapter
	Calendar *adapter.CalendarAdapter
	Task     *adapter.TaskAdapter
	IM       *adapter.IMAdapter
	Mail     *adapter.MailAdapter
	Meeting  *adapter.MeetingAdapter
	ReadOnly bool
	TTL      time.Duration
}

func NewOperations(cfg OperationsConfig) *Operations {
	return &Operations{
		cli:  cfg.CLI,
		tree: cfg.Tree, drive: cfg.Drive, wiki: cfg.Wiki,
		calendar: cfg.Calendar, task: cfg.Task, im: cfg.IM,
		mail: cfg.Mail, meeting: cfg.Meeting,
		controls: newControlStore(),
		readOnly: cfg.ReadOnly, refreshTTL: cfg.TTL,
	}
}

func (o *Operations) Tree() *Tree { return o.tree }

func (o *Operations) ReadDir(ctx context.Context, path string) ([]*VNode, error) {
	node := o.tree.Resolve(path)
	if node == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
	}

	if node == o.tree.Root() {
		return node.Children(), nil
	}

	if node.Kind == NodeKindControlDir {
		return o.listControlDir(node)
	}

	if !node.NeedsRefresh(o.refreshTTL) {
		return node.Children(), nil
	}

	list, err := o.fetchEntries(ctx, node)
	if err != nil {
		return nil, err
	}

	node.ClearChildren()
	node.Page = list.Page
	for _, e := range list.Entries {
		nt := NodeFile
		if e.IsDir || doctype.IsDirectory(e.Type) {
			nt = NodeDir
		}
		modTime := e.ModTime
		if modTime.IsZero() {
			modTime = time.Now()
		}
		child := &VNode{
			Name:        e.Name,
			Token:       e.Token,
			DocType:     e.Type,
			NodeType:    nt,
			Kind:        NodeKindResource,
			Domain:      node.Domain,
			Size:        e.Size,
			ModTime:     modTime,
			CreatedTime: e.CreatedTime,
			TargetPath:  pathJoin(node.Path(), e.Name),
			children:    make(map[string]*VNode),
		}
		node.AddChild(child)
	}
	o.ensureControlChildren(node)
	node.SetPopulated()

	return node.Children(), nil
}

func (o *Operations) Read(ctx context.Context, path string) ([]byte, error) {
	node, err := o.resolveNode(ctx, path)
	if err != nil {
		return nil, err
	}
	if node.Kind != NodeKindResource {
		return o.readControlNode(node)
	}
	return o.readContent(ctx, node)
}

func (o *Operations) Write(ctx context.Context, path string, data []byte) error {
	if o.readOnly {
		return fmt.Errorf("%w: filesystem mounted read-only", ErrReadOnly)
	}
	node, err := o.resolveNode(ctx, path)
	if err != nil {
		return err
	}
	if node.Kind != NodeKindResource {
		return o.writeControlNode(ctx, node, data)
	}
	return o.writeContent(ctx, node, data)
}

func (o *Operations) Create(ctx context.Context, path string) (*VNode, error) {
	if o.readOnly {
		return nil, fmt.Errorf("%w: filesystem mounted read-only", ErrReadOnly)
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: cannot create at root level", ErrUnsupported)
	}

	parentPath := strings.Join(parts[:len(parts)-1], "/")
	fileName := parts[len(parts)-1]

	parent := o.tree.Resolve(parentPath)
	if parent == nil {
		if _, err := o.ReadDir(ctx, parentPath); err == nil {
			parent = o.tree.Resolve(parentPath)
		}
	}
	if parent == nil {
		return nil, fmt.Errorf("%w: parent %s", ErrNotFound, parentPath)
	}
	if parent.Kind != NodeKindResource {
		return nil, fmt.Errorf("%w: cannot create under control path %s", ErrUnsupported, parentPath)
	}

	domain := parent.Domain
	if domain == "" {
		domain = o.domainFromPath(parent)
	}

	if domain != "drive" {
		return nil, fmt.Errorf("%w: create for domain %s", ErrUnsupported, domain)
	}

	dt := doctype.TypeDocx
	baseName := strings.TrimSuffix(fileName, ".md")

	token, err := o.drive.Create(ctx, parent.Token, baseName, dt, nil)
	if err != nil {
		return nil, err
	}

	child := &VNode{
		Name:       parts[len(parts)-1],
		Token:      token,
		DocType:    dt,
		NodeType:   NodeFile,
		Kind:       NodeKindResource,
		Domain:     domain,
		TargetPath: path,
		ModTime:    time.Now(),
		children:   make(map[string]*VNode),
	}
	parent.AddChild(child)
	return child, nil
}

func (o *Operations) Mkdir(ctx context.Context, path string) (*VNode, error) {
	if o.readOnly {
		return nil, fmt.Errorf("%w: filesystem mounted read-only", ErrReadOnly)
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: cannot create directory at root level", ErrUnsupported)
	}

	parentPath := "/" + strings.Join(parts[:len(parts)-1], "/")
	dirName := parts[len(parts)-1]
	parent, err := o.resolveNode(ctx, parentPath)
	if err != nil {
		return nil, err
	}
	if parent.Kind != NodeKindResource {
		return nil, fmt.Errorf("%w: cannot mkdir under control path %s", ErrUnsupported, parentPath)
	}
	domain := o.domainFromPath(parent)
	if domain != "drive" {
		return nil, fmt.Errorf("%w: mkdir for domain %s", ErrUnsupported, domain)
	}

	token, err := o.drive.Create(ctx, parent.Token, dirName, doctype.TypeFolder, nil)
	if err != nil {
		return nil, err
	}
	child := &VNode{
		Name:       dirName,
		Token:      token,
		DocType:    doctype.TypeFolder,
		NodeType:   NodeDir,
		Kind:       NodeKindResource,
		Domain:     domain,
		TargetPath: path,
		ModTime:    time.Now(),
		children:   make(map[string]*VNode),
	}
	parent.AddChild(child)
	o.ensureControlChildren(child)
	return child, nil
}

func (o *Operations) Remove(ctx context.Context, path string) error {
	if o.readOnly {
		return fmt.Errorf("%w: filesystem mounted read-only", ErrReadOnly)
	}
	node, err := o.resolveNode(ctx, path)
	if err != nil {
		return err
	}
	if node.Kind != NodeKindResource {
		return fmt.Errorf("%w: cannot remove control node %s", ErrUnsupported, path)
	}
	var removeErr error
	switch node.Domain {
	case "drive":
		removeErr = o.drive.Delete(ctx, node.Token, node.DocType)
	case "mail":
		removeErr = o.mail.Trash(ctx, node.Token)
	default:
		removeErr = fmt.Errorf("%w: remove for domain %s", ErrUnsupported, node.Domain)
	}
	if removeErr != nil {
		return removeErr
	}
	if parent := node.Parent(); parent != nil {
		parent.mu.Lock()
		delete(parent.children, node.Name)
		parent.mu.Unlock()
	}
	return nil
}

func (o *Operations) Rename(ctx context.Context, oldPath, newPath string) error {
	if o.readOnly {
		return fmt.Errorf("%w: filesystem mounted read-only", ErrReadOnly)
	}
	node, err := o.resolveNode(ctx, oldPath)
	if err != nil {
		return err
	}
	if node.Kind != NodeKindResource {
		return fmt.Errorf("%w: cannot rename control node %s", ErrUnsupported, oldPath)
	}

	oldParent := node.Parent()
	if oldParent == nil {
		return fmt.Errorf("%w: cannot rename root node", ErrUnsupported)
	}

	newParentPath := pathpkgDir(newPath)
	newParent, err := o.resolveNode(ctx, newParentPath)
	if err != nil {
		return err
	}
	if newParent.Kind != NodeKindResource {
		return fmt.Errorf("%w: cannot move into control path %s", ErrUnsupported, newParentPath)
	}
	newName := pathBase(newPath)

	if node.Domain != "drive" {
		return fmt.Errorf("%w: rename for domain %s", ErrUnsupported, node.Domain)
	}

	if newName != node.Name {
		return fmt.Errorf("%w: remote rename is not mapped for drive resources", ErrUnsupported)
	}

	if oldParent == newParent {
		return nil
	}

	if err := o.executeDriveMove(ctx, node, newParent); err != nil {
		return err
	}

	oldParent.mu.Lock()
	delete(oldParent.children, node.Name)
	oldParent.mu.Unlock()
	newParent.AddChild(node)
	return nil
}

func (o *Operations) fetchEntries(ctx context.Context, node *VNode) (doctype.ListResult, error) {
	domain := node.Domain
	if domain == "" {
		domain = o.domainFromPath(node)
	}

	switch domain {
	case "drive":
		if node.Token == "" {
			return o.drive.ListRoot(ctx)
		}
		if node.DocType != "" && node.DocType != doctype.TypeFolder {
			return o.drive.ListByType(ctx, node.Token, node.DocType)
		}
		return o.drive.ListFolder(ctx, node.Token)
	case "wiki":
		if node.Token == "" {
			return o.wiki.ListSpaces(ctx)
		}
		return o.wiki.ListNodes(ctx, node.Token)
	case "im":
		if node.Token == "" {
			return o.im.ListChats(ctx)
		}
		if strings.HasSuffix(node.Token, "|files") {
			chatID := strings.TrimSuffix(node.Token, "|files")
			return o.im.ListChatFiles(ctx, chatID)
		}
		return o.im.ListChatContents(ctx, node.Token)
	case "calendar":
		return o.calendar.ListEvents(ctx)
	case "tasks":
		return o.task.ListTasks(ctx)
	case "mail":
		if node.Token == "" {
			return o.mail.ListFolders(ctx)
		}
		return o.mail.ListMessages(ctx, node.Name)
	case "meetings":
		if node.Token == "" {
			return o.meeting.ListDateDirs(), nil
		}
		if datePattern.MatchString(node.Token) {
			return o.meeting.ListMeetings(ctx, node.Token)
		}
		return o.meeting.ListMeetingContents(node.Token), nil
	case "approval", "base", "contact", "docs", "minutes", "sheets", "vc", "_system":
		return staticDomainEntries(domain, node.Token), nil
	}

	return doctype.ListResult{}, fmt.Errorf("unknown domain: %s", domain)
}

func (o *Operations) readContent(ctx context.Context, node *VNode) ([]byte, error) {
	switch node.Domain {
	case "drive":
		return o.drive.Read(ctx, node.Token, node.DocType)
	case "wiki":
		return o.wiki.Read(ctx, node.Token)
	case "im":
		if strings.HasSuffix(node.Token, "|latest") {
			chatID := strings.TrimSuffix(node.Token, "|latest")
			return o.im.ReadMessages(ctx, chatID)
		}
		return nil, nil
	case "calendar":
		if node.Token == "_create" {
			return []byte("# New Event\n\nWrite event details here.\n"), nil
		}
		return o.calendar.ReadEvent(ctx, node.Token)
	case "tasks":
		if node.Token == "_create" {
			return []byte("# New Task\n\nWrite task summary here.\n"), nil
		}
		return o.task.ReadTask(ctx, node.Token)
	case "mail":
		return o.mail.ReadMessage(ctx, node.Token)
	case "meetings":
		return o.readMeetingContent(ctx, node)
	}
	return nil, fmt.Errorf("unsupported read: %s", node.Path())
}

func staticDomainEntries(domain, token string) doctype.ListResult {
	if token != "" {
		return doctype.ListResult{Page: doctype.PageInfo{SortKey: "fixed"}}
	}
	names := map[string][]string{
		"approval": {"instances", "tasks"},
		"base":     {"bases", "tables", "records", "fields", "views", "dashboards", "forms", "roles", "workflows"},
		"contact":  {"users", "search"},
		"docs":     {"search", "by-token", "media", "whiteboard"},
		"minutes":  {"minutes", "media"},
		"sheets":   {"spreadsheets", "sheets", "filters"},
		"vc":       {"meetings", "notes", "recordings"},
		"_system":  {"api", "schema", "auth", "config", "profile", "doctor", "event"},
	}
	entries := make([]doctype.Entry, 0, len(names[domain]))
	for _, name := range names[domain] {
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: domain + ":" + name,
			Type:  doctype.TypeFolder,
			IsDir: true,
		})
	}
	return doctype.ListResult{
		Entries: entries,
		Page: doctype.PageInfo{
			WindowSize: len(entries),
			SortKey:    "fixed",
		},
	}
}

func (o *Operations) writeContent(ctx context.Context, node *VNode, data []byte) error {
	switch node.Domain {
	case "drive":
		return o.drive.Write(ctx, node.Token, node.DocType, data)
	case "wiki":
		return o.wiki.Write(ctx, node.Token, data)
	case "im":
		if strings.HasSuffix(node.Token, "|send") {
			chatID := strings.TrimSuffix(node.Token, "|send")
			return o.im.SendMessage(ctx, chatID, data)
		}
		return fmt.Errorf("read-only")
	case "calendar":
		if node.Token == "_create" {
			return o.calendar.CreateEvent(ctx, data)
		}
		return fmt.Errorf("read-only")
	case "tasks":
		if node.Token == "_create" {
			return o.task.CreateTask(ctx, data)
		}
		return fmt.Errorf("read-only")
	}
	return fmt.Errorf("unsupported write: %s", node.Path())
}

func (o *Operations) readMeetingContent(ctx context.Context, node *VNode) ([]byte, error) {
	parts := strings.SplitN(node.Token, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid meeting token: %s", node.Token)
	}
	meetingID, part := parts[0], parts[1]

	switch part {
	case "meta":
		return o.meeting.ReadMeta(ctx, meetingID)
	case "summary":
		return o.meeting.ReadSummary(ctx, meetingID)
	case "transcript":
		return o.meeting.ReadTranscript(ctx, meetingID)
	case "todos":
		summary, err := o.meeting.ReadSummary(ctx, meetingID)
		if err != nil {
			return nil, err
		}
		return extractTodos(summary), nil
	case "recording":
		return o.meeting.ReadRecording(ctx, meetingID)
	}
	return nil, fmt.Errorf("unsupported meeting part: %s", part)
}

func (o *Operations) domainFromPath(node *VNode) string {
	for cur := node; cur != nil; cur = cur.parent {
		if cur.Domain != "" {
			return cur.Domain
		}
	}
	return ""
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

func pathJoin(parent, child string) string {
	if parent == "/" {
		return "/" + child
	}
	return parent + "/" + child
}

func pathBase(p string) string {
	parts := strings.Split(strings.TrimSuffix(p, "/"), "/")
	return parts[len(parts)-1]
}

func extractTodos(markdown []byte) []byte {
	var todos []string
	for _, line := range strings.Split(string(markdown), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") ||
			strings.Contains(strings.ToLower(trimmed), "todo") {
			todos = append(todos, line)
		}
	}
	if len(todos) == 0 {
		return []byte("No todos found.\n")
	}
	return []byte(strings.Join(todos, "\n") + "\n")
}

func (o *Operations) ExecuteOp(ctx context.Context, path string, payload []byte) ([]byte, error) {
	node, err := o.resolveNode(ctx, path)
	if err != nil {
		return nil, err
	}
	if node.Control != ControlRequestFile {
		return nil, fmt.Errorf("not an operation request node: %s", path)
	}
	if err := o.writeControlNode(ctx, node, payload); err != nil {
		return nil, err
	}
	resultPath := strings.Replace(path, ".request.json", ".result.json", 1)
	return o.controls.Get(resultPath), nil
}

func (o *Operations) RunQuery(ctx context.Context, path string, payload []byte) ([]byte, error) {
	return o.ExecuteOp(ctx, path, payload)
}

func (o *Operations) ListView(ctx context.Context, path string) ([]*VNode, error) {
	return o.ReadDir(ctx, path)
}
