package models

import (
	"sync"
	"time"
)

type UsageRecordManager struct {
	records map[string]*UsageRecord
	locks   map[string]*sync.Mutex
}

func (m *UsageRecordManager) Init() {
	m.records = make(map[string]*UsageRecord)
	m.locks = make(map[string]*sync.Mutex)
}

func (m *UsageRecordManager) RecordUsageFromProxyRequest(proxyReq *ApronServiceRequest, reqDetail *RequestDetail) {
	userKey := string(reqDetail.UserKey)
	recordKey := GenerateUsageRecordKey(proxyReq.ServiceId, userKey)
	rcd, ok := m.records[recordKey]
	if ok {
		m.locks[recordKey].Lock()
		defer m.locks[recordKey].Unlock()
		rcd.DoRecord(1, uint64(len(reqDetail.RequestBody)), 0)
	} else {
		m.locks[recordKey] = &sync.Mutex{}
		m.locks[recordKey].Lock()
		defer m.locks[recordKey].Unlock()

		currentTs := uint64(time.Now().UTC().Unix())
		m.records[recordKey] = &UsageRecord{
			AccountId:       userKey,
			ServiceUuid:     proxyReq.ServiceId,
			StartTimestamp:  currentTs,
			AccessCount:     1,
			UploadTraffic:   uint64(len(reqDetail.RequestBody)),
			DownloadTraffic: 0,
		}
	}
}

func (m *UsageRecordManager) ExportUsage(serviceId, userKey string) (string, error) {
	return "", nil
}

func (m *UsageRecordManager) ExportAllUsage() ([]*UsageRecord, error) {
	return nil, nil
}
