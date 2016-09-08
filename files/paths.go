package files

import (
  "strings"
  "path/filepath"
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
  cleanPath := filepath.Clean(path)
  return filepath.ToSlash(cleanPath)
}
