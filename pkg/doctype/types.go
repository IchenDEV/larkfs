package doctype

import (
	"context"
	"errors"
	"time"
)

var ErrReadOnly = errors.New("read-only document type")

type DocType string

const (
	TypeDocx     DocType = "docx"
	TypeDoc      DocType = "doc"
	TypeSheet    DocType = "sheet"
	TypeBitable  DocType = "bitable"
	TypeFile     DocType = "file"
	TypeFolder   DocType = "folder"
	TypeSlides   DocType = "slides"
	TypeMindnote DocType = "mindnote"
)

type Entry struct {
	Name        string
	Token       string
	Type        DocType
	Size        int64
	IsDir       bool
	ModTime     time.Time
	CreatedTime time.Time
}

type TypeHandler interface {
	IsDirectory() bool
	Extension() string
	List(ctx context.Context, token string) ([]Entry, error)
	Read(ctx context.Context, token string) ([]byte, error)
	Write(ctx context.Context, token string, data []byte) error
	Create(ctx context.Context, parentToken string, name string, data []byte) (string, error)
	Delete(ctx context.Context, token string) error
}

func IsReadOnly(t DocType) bool {
	switch t {
	case TypeDoc, TypeSlides, TypeMindnote:
		return true
	}
	return false
}

func IsDirectory(t DocType) bool {
	switch t {
	case TypeFolder, TypeSheet, TypeBitable:
		return true
	}
	return false
}

func FileExtension(t DocType) string {
	switch t {
	case TypeDocx, TypeDoc:
		return ".md"
	case TypeSheet:
		return ".sheet"
	case TypeBitable:
		return ".base"
	case TypeSlides:
		return ".slides.json"
	case TypeMindnote:
		return ".mindnote.json"
	default:
		return ""
	}
}
