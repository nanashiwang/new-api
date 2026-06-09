package common

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/static"
)

// Credit: https://github.com/gin-contrib/static/issues/19

type embedFileSystem struct {
	http.FileSystem
}

func (e *embedFileSystem) Exists(prefix string, path string) bool {
	path, ok := e.normalizeStaticPathForExists(prefix, path)
	if !ok {
		return false
	}
	file, err := e.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func (e *embedFileSystem) normalizeStaticPathForExists(prefix string, path string) (string, bool) {
	if prefix != "" {
		if prefix != "/" && path != prefix && !strings.HasPrefix(path, prefix+"/") {
			return "", false
		}
		path = strings.TrimPrefix(path, prefix)
		if path == "" {
			path = "/"
		}
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path, path != "/"
}

func (e *embedFileSystem) Open(name string) (http.File, error) {
	if name == "/" {
		// This will make sure the index page goes to NoRouter handler,
		// which will use the replaced index bytes with analytic codes.
		return nil, os.ErrNotExist
	}
	return e.FileSystem.Open(name)
}

func EmbedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	efs, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return &embedFileSystem{
		FileSystem: http.FS(efs),
	}
}
