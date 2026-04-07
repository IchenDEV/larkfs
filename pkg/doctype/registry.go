package doctype

import "github.com/IchenDEV/larkfs/pkg/cli"

type Registry struct {
	handlers map[DocType]TypeHandler
}

func NewRegistry(exec *cli.Executor, cacheDir string) *Registry {
	r := &Registry{handlers: make(map[DocType]TypeHandler)}
	r.handlers[TypeDocx] = NewDocxHandler(exec)
	r.handlers[TypeSheet] = NewSheetHandler(exec)
	r.handlers[TypeBitable] = NewBitableHandler(exec)
	r.handlers[TypeFile] = NewFileHandler(exec, cacheDir)
	r.handlers[TypeFolder] = NewFolderHandler(exec)
	r.handlers[TypeDoc] = NewReadonlyHandler(exec, TypeDoc)
	r.handlers[TypeSlides] = NewReadonlyHandler(exec, TypeSlides)
	r.handlers[TypeMindnote] = NewReadonlyHandler(exec, TypeMindnote)
	return r
}

func (r *Registry) Handler(t DocType) TypeHandler {
	if h, ok := r.handlers[t]; ok {
		return h
	}
	return r.handlers[TypeFile]
}
