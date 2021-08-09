package models

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type AggregatedAccessRecordManager struct {
	records map[string]*AggregatedAccessRecord
	locks   map[string]*sync.Mutex
}

func (m *AggregatedAccessRecordManager) Init() {
	m.records = make(map[string]*AggregatedAccessRecord)
	m.locks = make(map[string]*sync.Mutex)
}

func (m *AggregatedAccessRecordManager) IncUsage(serviceId, userKey string) {
	recordKey := AccessRecordStorageKeyFrom(serviceId, userKey)
	rcd, ok := m.records[recordKey]
	if ok {
		m.locks[recordKey].Lock()
		defer m.locks[recordKey].Unlock()
		rcd.Usage++
	} else {
		m.locks[recordKey] = &sync.Mutex{}
		m.locks[recordKey].Lock()
		defer m.locks[recordKey].Unlock()

		currentTs := uint64(time.Now().UTC().Unix())
		m.records[recordKey] = &AggregatedAccessRecord{
			Id:          currentTs,
			ServiceUuid: serviceId,
			UserKey:     userKey,
			StartTime:   currentTs,
			EndTime:     0,
			Usage:       1,
			PricePlan:   "",
			Cost:        0,
		}
	}
}

func (m *AggregatedAccessRecordManager) ExportUsage(serviceId, userKey string) (string, error) {
	recordKey := AccessRecordStorageKeyFrom(serviceId, userKey)
	rcd, ok := m.records[recordKey]
	if ok {
		return rcd.ExportStrAndFlush(), nil
	} else {
		return "", errors.New(fmt.Sprintf("no record found for service %s and user %s", serviceId, userKey))
	}
}

func (m *AggregatedAccessRecordManager) ExportAllUsage() ([]*AggregatedAccessRecord, error) {
	rslt := make([]*AggregatedAccessRecord, len(m.records))
	i := 0
	for _, r := range m.records {
		rslt[i] = r.ExportObjectAndFlush()
		i++
	}
	return rslt, nil
}
