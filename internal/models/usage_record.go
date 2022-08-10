package models

import (
	"fmt"
)

func (r *ApronUsageRecord) DoRecord(count uint64, upSize uint64, dwSize uint64) {
	r.AccessCount += count
	r.UploadTraffic += upSize
	r.DownloadTraffic += dwSize
}

func GenerateUsageRecordKey(serviceId, userKey string) string {
	return fmt.Sprintf("%s.%s", serviceId, userKey)
}
