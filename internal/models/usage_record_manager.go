package models

import (
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

type UsageRecordManager struct {
	records map[string]*ApronUsageRecord
	locks   map[string]*sync.RWMutex
}

func (m *UsageRecordManager) Init() {
	m.records = make(map[string]*ApronUsageRecord)
	m.locks = make(map[string]*sync.RWMutex)
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
		m.locks[recordKey] = &sync.RWMutex{}
		m.locks[recordKey].Lock()
		defer m.locks[recordKey].Unlock()

		m.records[recordKey] = &ApronUsageRecord{
			UserKey:         userKey,
			ServiceId:       proxyReq.ServiceId,
			StartTs:         time.Now().UnixMilli(),
			AccessCount:     1,
			UploadTraffic:   uint64(len(reqDetail.RequestBody)),
			DownloadTraffic: 0,
		}
	}
}

// TODO: Add method to record data from ApronServiceData

func (m *UsageRecordManager) ExportAllUsage(nodeId string) (NodeReport, error) {
	result := NodeReport{
		NodeId: nodeId,
	}
	for recordKey, record := range m.records {
		func() {
			m.locks[recordKey].Lock()
			defer m.locks[recordKey].Unlock()
			result.Records = append(result.Records, proto.Clone(record).(*ApronUsageRecord))
		}()
	}
	return result, nil
}
