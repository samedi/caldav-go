package files

import (
	"github.com/samedi/caldav-go/lib"
	"path/filepath"
	"strings"
)

const (
	Separator = string(filepath.Separator)
)

func AbsPath(path string) string {
	path = strings.Trim(path, "/")
	absPath, _ := filepath.Abs(path)

	return absPath
}

func DirPath(path string) string {
	return filepath.Dir(path)
}

func JoinPaths(paths ...string) string {
	return filepath.Join(paths...)
}

func ToSlashPath(path string) string {
	return lib.ToSlashPath(path)
}
