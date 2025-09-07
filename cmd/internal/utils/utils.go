package utils

import (
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func FormatEpoch(millis int64) string {
	return time.UnixMilli(millis).
		UTC().
		Format(time.RFC3339)
}

func NowUTC() int64 {
	return time.Now().
		UTC().
		UnixMilli()
}

func CheckFileExt(fileName string, valid []string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(fileName))
	return ext, slices.Contains(valid, ext)
}
