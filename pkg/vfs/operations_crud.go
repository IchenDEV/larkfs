package vfs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

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

	dt := createDocTypeForName(fileName)

	child := &VNode{
		Name:          parts[len(parts)-1],
		PendingCreate: dt == doctype.TypeFile,
		DocType:       dt,
		NodeType:      NodeFile,
		Kind:          NodeKindResource,
		Domain:        domain,
		TargetPath:    path,
		ModTime:       time.Now(),
		children:      make(map[string]*VNode),
	}
	if !child.PendingCreate {
		token, err := o.drive.Create(ctx, parent.Token, driveRemoteName(fileName, dt), dt, nil)
		if err != nil {
			return nil, err
		}
		child.Token = token
	}
	parent.AddChild(child)
	o.ensureResourceControlFiles(parent, child)
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
	if node.PendingCreate {
		if parent := node.Parent(); parent != nil {
			o.removeResourceControlFiles(parent, node)
			parent.mu.Lock()
			delete(parent.children, node.Name)
			parent.mu.Unlock()
		}
		return nil
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
		o.removeResourceControlFiles(parent, node)
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

	sameParent := oldParent == newParent
	sameName := newName == node.Name
	if sameParent && sameName {
		return nil
	}

	if node.PendingCreate {
		return o.relocateNode(node, oldParent, newParent, newName)
	}

	if !sameName {
		if err := o.drive.Rename(ctx, node.Token, driveRemoteName(newName, node.DocType)); err != nil {
			return err
		}
	}
	if !sameParent {
		if err := o.executeDriveMove(ctx, node, newParent); err != nil {
			return err
		}
	}
	return o.relocateNode(node, oldParent, newParent, newName)
}
