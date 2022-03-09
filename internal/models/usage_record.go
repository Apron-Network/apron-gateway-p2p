package models

import "fmt"

type UsageRecord struct {
	AccountId       string `json:"account_id"`
	ServiceUuid     string `json:"service_uuid"`
	StartTimestamp  uint64 `json:"start_timestamp"`
	EndTimestamp    uint64 `json:"end_timestamp"`
	AccessCount     uint64 `json:"access_count"`
	UploadTraffic   uint64 `json:"upload_traffic"`
	DownloadTraffic uint64 `json:"download_traffic"`
}

func (r *UsageRecord) DoRecord(count uint64, upSize uint64, dwSize uint64) {
	r.AccessCount += count
	r.UploadTraffic += upSize
	r.DownloadTraffic += dwSize
}

func GenerateUsageRecordKey(serviceId, userKey string) string {
	return fmt.Sprintf("%s.%s", serviceId, userKey)
}
