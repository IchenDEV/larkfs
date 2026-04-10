package vfs

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/adapter"
	"github.com/IchenDEV/larkfs/pkg/doctype"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Operations struct {
	tree       *Tree
	drive      *adapter.DriveAdapter
	wiki       *adapter.WikiAdapter
	calendar   *adapter.CalendarAdapter
	task       *adapter.TaskAdapter
	im         *adapter.IMAdapter
	mail       *adapter.MailAdapter
	meeting    *adapter.MeetingAdapter
	readOnly   bool
	refreshTTL time.Duration
}

type OperationsConfig struct {
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
		tree: cfg.Tree, drive: cfg.Drive, wiki: cfg.Wiki,
		calendar: cfg.Calendar, task: cfg.Task, im: cfg.IM,
		mail: cfg.Mail, meeting: cfg.Meeting,
		readOnly: cfg.ReadOnly, refreshTTL: cfg.TTL,
	}
}

func (o *Operations) Tree() *Tree { return o.tree }

func (o *Operations) ReadDir(ctx context.Context, path string) ([]*VNode, error) {
	node := o.tree.Resolve(path)
	if node == nil {
		return nil, fmt.Errorf("not found: %s", path)
	}

	if node == o.tree.Root() {
		return node.Children(), nil
	}

	if !node.NeedsRefresh(o.refreshTTL) {
		return node.Children(), nil
	}

	entries, err := o.fetchEntries(ctx, node)
	if err != nil {
		return nil, err
	}

	node.ClearChildren()
	for _, e := range entries {
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
			Domain:      node.Domain,
			Size:        e.Size,
			ModTime:     modTime,
			CreatedTime: e.CreatedTime,
			children:    make(map[string]*VNode),
		}
		node.AddChild(child)
	}
	node.SetPopulated()

	return node.Children(), nil
}

func (o *Operations) Read(ctx context.Context, path string) ([]byte, error) {
	node := o.tree.Resolve(path)
	if node == nil {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return o.readContent(ctx, node)
}

func (o *Operations) Write(ctx context.Context, path string, data []byte) error {
	if o.readOnly {
		return fmt.Errorf("filesystem mounted read-only")
	}
	node := o.tree.Resolve(path)
	if node == nil {
		return fmt.Errorf("not found: %s", path)
	}
	return o.writeContent(ctx, node, data)
}

func (o *Operations) Create(ctx context.Context, path string) (*VNode, error) {
	if o.readOnly {
		return nil, fmt.Errorf("filesystem mounted read-only")
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("cannot create at root level")
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
		return nil, fmt.Errorf("parent not found: %s", parentPath)
	}

	domain := parent.Domain
	if domain == "" {
		domain = o.domainFromPath(parent)
	}

	if domain != "drive" {
		return nil, fmt.Errorf("create not supported for domain: %s", domain)
	}

	dt := doctype.TypeDocx
	fileName = strings.TrimSuffix(fileName, ".md")

	token, err := o.drive.Create(ctx, parent.Token, fileName, dt, nil)
	if err != nil {
		return nil, err
	}

	child := &VNode{
		Name:     parts[len(parts)-1],
		Token:    token,
		DocType:  dt,
		NodeType: NodeFile,
		Domain:   domain,
		ModTime:  time.Now(),
		children: make(map[string]*VNode),
	}
	parent.AddChild(child)
	return child, nil
}

func (o *Operations) fetchEntries(ctx context.Context, node *VNode) ([]doctype.Entry, error) {
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
	}

	return nil, fmt.Errorf("unknown domain: %s", domain)
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
