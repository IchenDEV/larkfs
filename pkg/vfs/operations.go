package vfs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/adapter"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
)

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
	cacheDir   string
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
	CacheDir string
}

func NewOperations(cfg OperationsConfig) *Operations {
	return &Operations{
		cli:  cfg.CLI,
		tree: cfg.Tree, drive: cfg.Drive, wiki: cfg.Wiki,
		calendar: cfg.Calendar, task: cfg.Task, im: cfg.IM,
		mail: cfg.Mail, meeting: cfg.Meeting,
		controls: newControlStore(),
		readOnly: cfg.ReadOnly, refreshTTL: cfg.TTL,
		cacheDir: cfg.CacheDir,
	}
}

func (o *Operations) Tree() *Tree { return o.tree }

func (o *Operations) Stat(ctx context.Context, path string) (*VNode, error) {
	return o.resolveNode(ctx, path)
}

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
		child := newVNodeNow(&VNode{
			Name:        e.Name,
			Token:       e.Token,
			DocType:     e.Type,
			NodeType:    nt,
			Kind:        NodeKindResource,
			Domain:      node.Domain,
			CreatedTime: e.CreatedTime,
			TargetPath:  pathJoin(node.Path(), e.Name),
		})
		child.SetSize(e.Size)
		child.SetModTime(modTime)
		node.AddChild(child)
		o.ensureResourceControlFiles(node, child)
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
