package vfs

import (
	"testing"
	"time"
)

var allDomains = []string{"drive", "wiki", "im", "calendar", "tasks", "mail", "meetings"}

func TestTreeResolve(t *testing.T) {
	tree := NewTree(allDomains)

	root := tree.Resolve("/")
	if root == nil {
		t.Fatal("root should not be nil")
	}

	drive := tree.Resolve("/drive")
	if drive == nil {
		t.Fatal("/drive should not be nil")
	}
	if drive.Name != "drive" {
		t.Errorf("expected name=drive, got %s", drive.Name)
	}

	nonexist := tree.Resolve("/nonexistent")
	if nonexist != nil {
		t.Error("/nonexistent should be nil")
	}
}

func TestTreeFilteredDomains(t *testing.T) {
	tree := NewTree([]string{"drive", "wiki"})

	drive := tree.Resolve("/drive")
	if drive == nil {
		t.Fatal("/drive should exist")
	}

	im := tree.Resolve("/im")
	if im != nil {
		t.Error("/im should not exist when filtered out")
	}

	children := tree.Root().Children()
	if len(children) != 2 {
		t.Errorf("expected 2 domains, got %d", len(children))
	}
}

func TestVNodePath(t *testing.T) {
	tree := NewTree(allDomains)
	drive := tree.Resolve("/drive")

	child := &VNode{
		Name:     "myfile.md",
		NodeType: NodeFile,
		children: make(map[string]*VNode),
	}
	drive.AddChild(child)

	if child.Path() != "/drive/myfile.md" {
		t.Errorf("expected /drive/myfile.md, got %s", child.Path())
	}
}

func TestVNodePathDeep(t *testing.T) {
	tree := NewTree(allDomains)
	drive := tree.Resolve("/drive")

	folderA := &VNode{Name: "a", NodeType: NodeDir, children: make(map[string]*VNode)}
	drive.AddChild(folderA)
	folderB := &VNode{Name: "b", NodeType: NodeDir, children: make(map[string]*VNode)}
	folderA.AddChild(folderB)
	file := &VNode{Name: "c.md", NodeType: NodeFile, children: make(map[string]*VNode)}
	folderB.AddChild(file)

	got := file.Path()
	want := "/drive/a/b/c.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewTreeRootIsPopulated(t *testing.T) {
	tree := NewTree(allDomains)
	root := tree.Root()

	if root.NeedsRefresh(time.Second) {
		t.Error("root node should be populated after NewTree")
	}
}

func TestVNodeNeedsRefresh(t *testing.T) {
	node := &VNode{
		NodeType: NodeDir,
		children: make(map[string]*VNode),
	}

	if !node.NeedsRefresh(time.Second) {
		t.Error("unpopulated node should need refresh")
	}

	node.AddChild(&VNode{Name: "a", children: make(map[string]*VNode)})
	node.SetPopulated()

	if node.NeedsRefresh(time.Second) {
		t.Error("just-populated node should not need refresh")
	}

	time.Sleep(50 * time.Millisecond)
	if !node.NeedsRefresh(10 * time.Millisecond) {
		t.Error("node should need refresh after TTL")
	}
}

func TestVNodeNeedsRefreshEmptyDir(t *testing.T) {
	node := &VNode{
		NodeType: NodeDir,
		children: make(map[string]*VNode),
	}
	node.SetPopulated()

	if node.NeedsRefresh(time.Second) {
		t.Error("populated empty dir should not need refresh within TTL")
	}

	time.Sleep(50 * time.Millisecond)
	if !node.NeedsRefresh(10 * time.Millisecond) {
		t.Error("populated empty dir should need refresh after TTL")
	}
}

func TestNewTreeCalendarHasCreateNode(t *testing.T) {
	tree := NewTree(allDomains)

	node := tree.Resolve("/calendar/_create.md")
	if node == nil {
		t.Fatal("calendar/_create.md should exist at init")
	}
	if node.Token != "_create" {
		t.Errorf("expected token=_create, got %s", node.Token)
	}
	if node.IsDir() {
		t.Error("_create.md should be a file, not a directory")
	}
	if node.Domain != "calendar" {
		t.Errorf("expected domain=calendar, got %s", node.Domain)
	}
}

func TestNewTreeTasksHasCreateNode(t *testing.T) {
	tree := NewTree(allDomains)

	node := tree.Resolve("/tasks/_create.md")
	if node == nil {
		t.Fatal("tasks/_create.md should exist at init")
	}
	if node.Token != "_create" {
		t.Errorf("expected token=_create, got %s", node.Token)
	}
	if node.IsDir() {
		t.Error("_create.md should be a file, not a directory")
	}
}

func TestNewTreeDriveNoCreateNode(t *testing.T) {
	tree := NewTree([]string{"drive"})

	node := tree.Resolve("/drive/_create.md")
	if node != nil {
		t.Error("drive should not have _create.md")
	}
}

func TestVNodeCreatedTime(t *testing.T) {
	created := time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC)
	node := &VNode{
		Name:        "test.md",
		NodeType:    NodeFile,
		CreatedTime: created,
		children:    make(map[string]*VNode),
	}
	if !node.CreatedTime.Equal(created) {
		t.Errorf("expected CreatedTime=%v, got %v", created, node.CreatedTime)
	}

	zero := &VNode{Name: "zero.md", NodeType: NodeFile, children: make(map[string]*VNode)}
	if !zero.CreatedTime.IsZero() {
		t.Error("expected zero CreatedTime for default VNode")
	}
}
