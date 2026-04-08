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
