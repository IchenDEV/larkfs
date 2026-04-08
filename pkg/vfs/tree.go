package vfs

import (
	"strings"
	"sync"
	"time"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

type NodeType int

const (
	NodeDir NodeType = iota
	NodeFile
)

type VNode struct {
	Name     string
	Token    string
	DocType  doctype.DocType
	NodeType NodeType
	Domain   string
	Size     int64
	ModTime  time.Time

	mu          sync.RWMutex
	children    map[string]*VNode
	parent      *VNode
	populatedAt time.Time
}

func NewRootNode() *VNode {
	return &VNode{
		Name:     "",
		NodeType: NodeDir,
		children: make(map[string]*VNode),
		ModTime:  time.Now(),
	}
}

func (n *VNode) AddChild(child *VNode) {
	n.mu.Lock()
	defer n.mu.Unlock()
	child.parent = n
	n.children[child.Name] = child
}

func (n *VNode) GetChild(name string) *VNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.children[name]
}

func (n *VNode) Children() []*VNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	result := make([]*VNode, 0, len(n.children))
	for _, c := range n.children {
		result = append(result, c)
	}
	return result
}

func (n *VNode) ClearChildren() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.children = make(map[string]*VNode)
}

func (n *VNode) SetPopulated() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.populatedAt = time.Now()
}

func (n *VNode) NeedsRefresh(ttl time.Duration) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if len(n.children) == 0 {
		return true
	}
	return time.Since(n.populatedAt) > ttl
}

func (n *VNode) IsDir() bool {
	return n.NodeType == NodeDir
}

func (n *VNode) Parent() *VNode { return n.parent }

func (n *VNode) Path() string {
	if n.parent == nil {
		return "/"
	}
	var parts []string
	for cur := n; cur != nil && cur.parent != nil; cur = cur.parent {
		parts = append(parts, cur.Name)
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return "/" + strings.Join(parts, "/")
}

type Tree struct {
	root *VNode
}

func NewTree(domains []string) *Tree {
	root := NewRootNode()
	for _, domain := range domains {
		root.AddChild(&VNode{
			Name:     domain,
			NodeType: NodeDir,
			Domain:   domain,
			children: make(map[string]*VNode),
			ModTime:  time.Now(),
		})
	}
	root.SetPopulated()
	return &Tree{root: root}
}

func (t *Tree) Root() *VNode {
	return t.root
}

func (t *Tree) Resolve(path string) *VNode {
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "." {
		return t.root
	}

	parts := strings.Split(path, "/")
	node := t.root
	for _, p := range parts {
		if p == "" {
			continue
		}
		child := node.GetChild(p)
		if child == nil {
			return nil
		}
		node = child
	}
	return node
}

func (t *Tree) DomainNode(domain string) *VNode {
	return t.root.GetChild(domain)
}
