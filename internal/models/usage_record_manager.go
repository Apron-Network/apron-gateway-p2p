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
	userKey, serviceId := ExtractServiceInfoFromRequestID(proxyReq.RequestId)
	recordKey := GenerateUsageRecordKey(serviceId, userKey)
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
			ServiceId:       serviceId,
			StartTs:         time.Now().UnixMilli(),
			AccessCount:     1,
			UploadTraffic:   uint64(len(reqDetail.RequestBody)),
			DownloadTraffic: 0,
		}
	}
}

func (m *UsageRecordManager) RecordUsageFromProxyData(proxyData *ApronServiceData, isClientToService bool) {
	userKey, serviceId := ExtractServiceInfoFromRequestID(proxyData.RequestId)
	recordKey := GenerateUsageRecordKey(serviceId, userKey)

	// In this function, the record should have already created in records, so access it directly
	rcd, _ := m.records[recordKey]
	m.locks[recordKey].Lock()
	defer m.locks[recordKey].Unlock()
	if isClientToService {
		rcd.DoRecord(0, uint64(len(proxyData.RawData)), 0)
	} else {
		rcd.DoRecord(0, 0, uint64(len(proxyData.RawData)))
	}
}

// TODO: Add method to record data from ApronServiceData

func (m *UsageRecordManager) ExportAllUsage(nodeId string) (NodeReport, error) {
	result := NodeReport{
		NodeId: nodeId,
	}
	for recordKey, lock := range m.locks {
		func() {
			lock.Lock()
			defer lock.Unlock()

			tmpRecord := proto.Clone(m.records[recordKey]).(*ApronUsageRecord)
			tmpRecord.EndTs = time.Now().UnixMilli()
			result.Records = append(result.Records, tmpRecord)
			delete(m.records, recordKey)
			delete(m.locks, recordKey)
		}()
	}
	return result, nil
}
