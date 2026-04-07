package model

import "strings"

func profitBoardUsesWalletObserver(config ProfitBoardTokenPricingConfig) bool {
	return strings.TrimSpace(config.UpstreamMode) == ProfitBoardUpstreamModeWallet
}

func profitBoardComboUsesWalletObserver(config profitBoardResolvedComboPricing) bool {
	return strings.TrimSpace(config.UpstreamMode) == ProfitBoardUpstreamModeWallet && config.UpstreamAccountID > 0
}

func profitBoardHasWalletObserverCombo(comboPricingMap map[string]profitBoardResolvedComboPricing) bool {
	for _, config := range comboPricingMap {
		if profitBoardComboUsesWalletObserver(config) {
			return true
		}
	}
	return false
}

func profitBoardWalletObserverCombosByAccount(comboPricingMap map[string]profitBoardResolvedComboPricing) map[int][]string {
	accountCombos := make(map[int][]string)
	for comboID, config := range comboPricingMap {
		if !profitBoardComboUsesWalletObserver(config) {
			continue
		}
		accountCombos[config.UpstreamAccountID] = append(accountCombos[config.UpstreamAccountID], comboID)
	}
	return accountCombos
}

func applyProfitBoardObservedWalletCost(report *ProfitBoardReport, aggregate *profitBoardUpstreamAccountObservedAggregate, batches []ProfitBoardBatchInfo, comboIDs []string, granularity string, customIntervalMinutes int) {
	if report == nil || aggregate == nil {
		return
	}
	report.Warnings = append(report.Warnings, aggregate.Warnings...)
	if report.Summary.RequestCount <= 0 || aggregate.TotalCostUSD <= 0 {
		return
	}

	batchByID := make(map[string]ProfitBoardBatchInfo, len(batches))
	for _, batch := range batches {
		batchByID[batch.Id] = batch
	}

	comboIDSet := make(map[string]ProfitBoardBatchInfo, len(comboIDs))
	for _, comboID := range comboIDs {
		comboID = strings.TrimSpace(comboID)
		if comboID == "" {
			continue
		}
		batch, ok := batchByID[comboID]
		if !ok {
			continue
		}
		comboIDSet[comboID] = batch
	}
	if len(comboIDSet) == 0 {
		return
	}

	appliedTotalCostUSD := 0.0
	for _, observedPoint := range aggregate.Points {
		activeBatchIDs := make([]string, 0, len(comboIDSet))
		for comboID, batch := range comboIDSet {
			if batch.CreatedAt > 0 && observedPoint.SyncedAt < batch.CreatedAt {
				continue
			}
			activeBatchIDs = append(activeBatchIDs, comboID)
		}
		if len(activeBatchIDs) == 0 || observedPoint.CostUSD <= 0 {
			continue
		}

		bucketTimestamp, bucketLabel := buildProfitBoardBucket(observedPoint.SyncedAt, granularity, customIntervalMinutes)
		batchRequests := make(map[string]int, len(activeBatchIDs))
		totalBucketRequests := 0
		for _, point := range report.Timeseries {
			if point.BucketTimestamp != bucketTimestamp {
				continue
			}
			if _, ok := comboIDSet[point.BatchId]; !ok {
				continue
			}
			batchRequests[point.BatchId] += point.RequestCount
			totalBucketRequests += point.RequestCount
		}
		if totalBucketRequests <= 0 {
			continue
		}

		appliedTotalCostUSD += observedPoint.CostUSD
		for index := range report.BatchSummaries {
			summary := &report.BatchSummaries[index]
			requests := batchRequests[summary.BatchId]
			if requests <= 0 {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, requests)
			summary.RemoteObservedCostUSD += share
			summary.UpstreamCostUSD += share
			summary.ConfiguredProfitUSD -= share
			summary.ActualProfitUSD -= share
		}

		for index := range report.Timeseries {
			point := &report.Timeseries[index]
			requests := batchRequests[point.BatchId]
			if requests <= 0 || point.BucketTimestamp != bucketTimestamp {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, point.RequestCount)
			point.RemoteObservedCostUSD += share
			point.UpstreamCostUSD += share
			point.ConfiguredProfitUSD -= share
			point.ActualProfitUSD -= share
		}

		for _, batchID := range activeBatchIDs {
			if batchRequests[batchID] > 0 {
				continue
			}
			batch := comboIDSet[batchID]
			_ = getOrCreateProfitBoardTimeseriesPoint(report, batchID, batch.Name, bucketTimestamp, bucketLabel)
		}

		for index := range report.ChannelBreakdown {
			item := &report.ChannelBreakdown[index]
			requests := batchRequests[item.BatchId]
			if requests <= 0 {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, item.RequestCount)
			item.UpstreamCostUSD += share
			item.ConfiguredProfitUSD -= share
			item.ActualProfitUSD -= share
		}
		for index := range report.ModelBreakdown {
			item := &report.ModelBreakdown[index]
			requests := batchRequests[item.BatchId]
			if requests <= 0 {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, item.RequestCount)
			item.UpstreamCostUSD += share
			item.ConfiguredProfitUSD -= share
			item.ActualProfitUSD -= share
		}
	}
	report.Summary.RemoteObservedCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostUSD += appliedTotalCostUSD
	report.Summary.ConfiguredProfitUSD -= appliedTotalCostUSD
	report.Summary.ActualProfitUSD -= appliedTotalCostUSD
}
