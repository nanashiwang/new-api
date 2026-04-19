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

func applyProfitBoardObservedWalletCost(report *ProfitBoardReport, aggregate *profitBoardUpstreamAccountObservedAggregate, comboPricingMap map[string]profitBoardResolvedComboPricing, batches []ProfitBoardBatchInfo, comboIDs []string, granularity string, customIntervalMinutes int) {
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
	appliedTotalCostCNY := 0.0
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
			// Fallback：活跃 combo 在此桶内没有本站日志请求（例如 wallet observer 走上游扣费、本站无 logs），
			// 按 combo 数均分成本到 timeseries / batchSummary，保证 trend 图上游费用与 overview 口径一致。
			activeCount := len(activeBatchIDs)
			if activeCount == 0 {
				continue
			}
			shareUSD := observedPoint.CostUSD / float64(activeCount)
			appliedTotalCostUSD += observedPoint.CostUSD
			for _, batchID := range activeBatchIDs {
				batch := comboIDSet[batchID]
				shareCNY := profitBoardConfiguredUpstreamCostCNY(shareUSD, comboPricingMap[batchID])
				appliedTotalCostCNY += shareCNY

				point := getOrCreateProfitBoardTimeseriesPoint(report, batchID, batch.Name, bucketTimestamp, bucketLabel)
				point.RemoteObservedCostUSD += shareUSD
				point.UpstreamCostUSD += shareUSD
				point.UpstreamCostCNY += shareCNY
				point.ConfiguredProfitUSD -= shareUSD
				point.ConfiguredProfitCNY -= shareCNY
				point.ActualProfitUSD -= shareUSD

				for index := range report.BatchSummaries {
					summary := &report.BatchSummaries[index]
					if summary.BatchId != batchID {
						continue
					}
					summary.RemoteObservedCostUSD += shareUSD
					summary.UpstreamCostUSD += shareUSD
					summary.UpstreamCostCNY += shareCNY
					summary.ConfiguredProfitUSD -= shareUSD
					summary.ConfiguredProfitCNY -= shareCNY
					summary.ActualProfitUSD -= shareUSD
					break
				}
			}
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
			shareCNY := profitBoardConfiguredUpstreamCostCNY(share, comboPricingMap[summary.BatchId])
			summary.RemoteObservedCostUSD += share
			summary.UpstreamCostUSD += share
			summary.UpstreamCostCNY += shareCNY
			summary.ConfiguredProfitUSD -= share
			summary.ConfiguredProfitCNY -= shareCNY
			summary.ActualProfitUSD -= share
			appliedTotalCostCNY += shareCNY
		}

		for index := range report.Timeseries {
			point := &report.Timeseries[index]
			requests := batchRequests[point.BatchId]
			if requests <= 0 || point.BucketTimestamp != bucketTimestamp {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, point.RequestCount)
			shareCNY := profitBoardConfiguredUpstreamCostCNY(share, comboPricingMap[point.BatchId])
			point.RemoteObservedCostUSD += share
			point.UpstreamCostUSD += share
			point.UpstreamCostCNY += shareCNY
			point.ConfiguredProfitUSD -= share
			point.ConfiguredProfitCNY -= shareCNY
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
			shareCNY := profitBoardConfiguredUpstreamCostCNY(share, comboPricingMap[item.BatchId])
			item.UpstreamCostUSD += share
			item.UpstreamCostCNY += shareCNY
			item.ConfiguredProfitUSD -= share
			item.ConfiguredProfitCNY -= shareCNY
			item.ActualProfitUSD -= share
		}
		for index := range report.ModelBreakdown {
			item := &report.ModelBreakdown[index]
			requests := batchRequests[item.BatchId]
			if requests <= 0 {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, item.RequestCount)
			shareCNY := profitBoardConfiguredUpstreamCostCNY(share, comboPricingMap[item.BatchId])
			item.UpstreamCostUSD += share
			item.UpstreamCostCNY += shareCNY
			item.ConfiguredProfitUSD -= share
			item.ConfiguredProfitCNY -= shareCNY
			item.ActualProfitUSD -= share
		}
	}
	report.Summary.RemoteObservedCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostCNY += appliedTotalCostCNY
	report.Summary.ConfiguredProfitUSD -= appliedTotalCostUSD
	report.Summary.ConfiguredProfitCNY -= appliedTotalCostCNY
	report.Summary.ActualProfitUSD -= appliedTotalCostUSD
}
