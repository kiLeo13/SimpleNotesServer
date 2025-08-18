package service

import (
	"time"
)

func FormatEpoch(millis int64) string {
	return time.UnixMilli(millis).
		UTC().
		Format(time.RFC3339)
}
