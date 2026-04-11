package vfs

import (
	"sync"
	"time"
)

type controlStore struct {
	mu      sync.RWMutex
	content map[string][]byte
}

func newControlStore() *controlStore {
	return &controlStore{content: make(map[string][]byte)}
}

func (s *controlStore) Get(path string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if data, ok := s.content[path]; ok {
		return append([]byte(nil), data...)
	}
	return nil
}

func (s *controlStore) Set(path string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.content[path] = append([]byte(nil), data...)
}

func newControlFile(parent *VNode, name string, control ControlKind, action string) *VNode {
	return &VNode{
		Name:       name,
		NodeType:   NodeFile,
		Kind:       NodeKindControl,
		Control:    control,
		Domain:     parent.Domain,
		TargetPath: parent.TargetPath,
		Action:     action,
		ModTime:    time.Now(),
		children:   make(map[string]*VNode),
	}
}

func newControlDir(parent *VNode, name string, control ControlKind, action string) *VNode {
	return &VNode{
		Name:       name,
		NodeType:   NodeDir,
		Kind:       NodeKindControlDir,
		Control:    control,
		Domain:     parent.Domain,
		TargetPath: parent.TargetPath,
		Action:     action,
		ModTime:    time.Now(),
		children:   make(map[string]*VNode),
	}
}
