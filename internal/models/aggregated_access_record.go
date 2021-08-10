package models

import (
	"apron.network/gateway-p2p/internal"
	"encoding/json"
	"fmt"
	"time"
)

type AggregatedAccessRecord struct {
	Id          uint64 `json:"id"`
	ServiceUuid string `json:"service_uuid"`
	UserKey     string `json:"user_key"`
	StartTime   uint64 `json:"start_time"`
	EndTime     uint64 `json:"end_time"`
	Usage       uint64 `json:"usage"`
	PricePlan   string `json:"price_plan"`
	Cost        uint64 `json:"Cost"`
}

func (r *AggregatedAccessRecord) Reset(startTime time.Time) {
	epochSecond := uint64(startTime.Unix())

	r.Id = epochSecond // TODO: Confirm how to generate the ID
	r.StartTime = epochSecond
	r.Usage = 0
}

func (r *AggregatedAccessRecord) ExportStrAndFlush() string {
	currentTime := time.Now().UTC()

	r.EndTime = uint64(currentTime.Unix())

	strData, err := json.Marshal(r)
	internal.CheckError(err)

	r.Reset(currentTime)

	return string(strData)
}

func (r *AggregatedAccessRecord) ExportObjectAndFlush() *AggregatedAccessRecord {
	currentTime := time.Now().UTC()

	r.EndTime = uint64(currentTime.Unix())
	rslt := &AggregatedAccessRecord{}
	tmpData, err := json.Marshal(r)
	internal.CheckError(err)

	r.Reset(currentTime)
	err = json.Unmarshal(tmpData, rslt)
	internal.CheckError(err)
	return rslt
}

func AccessRecordStorageKeyFrom(serviceUuid, userKey string) string {
	return fmt.Sprintf("%s.%s", serviceUuid, userKey)
}
