package model

func profitBoardEffectiveStartTimestamp(batch ProfitBoardBatchInfo, startTimestamp int64) int64 {
	if batch.CreatedAt > 0 && batch.CreatedAt > startTimestamp {
		return batch.CreatedAt
	}
	return startTimestamp
}

func profitBoardFixedAllocationShare(total float64, totalRequests int, itemRequests int) float64 {
	if total == 0 || totalRequests <= 0 || itemRequests <= 0 {
		return 0
	}
	return total * float64(itemRequests) / float64(totalRequests)
}

func profitBoardTimestampInRange(timestamp int64, startTimestamp int64, endTimestamp int64) bool {
	if timestamp <= 0 {
		return false
	}
	if startTimestamp > 0 && timestamp < startTimestamp {
		return false
	}
	if endTimestamp > 0 && timestamp > endTimestamp {
		return false
	}
	return true
}

func profitBoardBatchCreatedAtMap(batches []ProfitBoardBatchInfo) map[string]int64 {
	result := make(map[string]int64, len(batches))
	for _, batch := range batches {
		result[batch.Id] = batch.CreatedAt
	}
	return result
}

func getOrCreateProfitBoardTimeseriesPoint(report *ProfitBoardReport, batchID string, batchName string, bucketTimestamp int64, bucketLabel string) *ProfitBoardTimeseriesPoint {
	for index := range report.Timeseries {
		point := &report.Timeseries[index]
		if point.BatchId == batchID && point.BucketTimestamp == bucketTimestamp {
			if point.Bucket == "" {
				point.Bucket = bucketLabel
			}
			if point.BatchName == "" {
				point.BatchName = batchName
			}
			return point
		}
	}
	report.Timeseries = append(report.Timeseries, ProfitBoardTimeseriesPoint{
		BatchId:         batchID,
		BatchName:       batchName,
		Bucket:          bucketLabel,
		BucketTimestamp: bucketTimestamp,
	})
	return &report.Timeseries[len(report.Timeseries)-1]
}
