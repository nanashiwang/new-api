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

func applyWalletCostToBatchSummary(summary *ProfitBoardBatchSummary, shareUSD float64, shareCNY float64) {
	summary.RemoteObservedCostUSD += shareUSD
	summary.UpstreamCostUSD += shareUSD
	summary.UpstreamCostCNY += shareCNY
	summary.ConfiguredProfitUSD -= shareUSD
	summary.ConfiguredProfitCNY -= shareCNY
	summary.ActualProfitUSD -= shareUSD
}

func applyWalletCostToTimeseriesPoint(point *ProfitBoardTimeseriesPoint, shareUSD float64, shareCNY float64) {
	point.RemoteObservedCostUSD += shareUSD
	point.UpstreamCostUSD += shareUSD
	point.UpstreamCostCNY += shareCNY
	point.ConfiguredProfitUSD -= shareUSD
	point.ConfiguredProfitCNY -= shareCNY
	point.ActualProfitUSD -= shareUSD
}

func applyWalletCostToBreakdownItem(item *ProfitBoardBreakdownItem, shareUSD float64, shareCNY float64) {
	item.UpstreamCostUSD += shareUSD
	item.UpstreamCostCNY += shareCNY
	item.ConfiguredProfitUSD -= shareUSD
	item.ConfiguredProfitCNY -= shareCNY
	item.ActualProfitUSD -= shareUSD
}

func applyProfitBoardObservedWalletCost(report *ProfitBoardReport, aggregate *profitBoardUpstreamAccountObservedAggregate, comboPricingMap map[string]profitBoardResolvedComboPricing, batches []ProfitBoardBatchInfo, comboIDs []string, granularity string, customIntervalMinutes int, cumulative bool) {
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

	if cumulative {
		applyProfitBoardObservedWalletCostCumulative(report, aggregate, comboPricingMap, comboIDSet)
		return
	}

	applyProfitBoardObservedWalletCostPerBucket(report, aggregate, comboPricingMap, comboIDSet, granularity, customIntervalMinutes)
}

func applyProfitBoardObservedWalletCostPerBucket(report *ProfitBoardReport, aggregate *profitBoardUpstreamAccountObservedAggregate, comboPricingMap map[string]profitBoardResolvedComboPricing, comboIDSet map[string]ProfitBoardBatchInfo, granularity string, customIntervalMinutes int) {
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
				applyWalletCostToTimeseriesPoint(point, shareUSD, shareCNY)

				for index := range report.BatchSummaries {
					summary := &report.BatchSummaries[index]
					if summary.BatchId != batchID {
						continue
					}
					applyWalletCostToBatchSummary(summary, shareUSD, shareCNY)
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
			applyWalletCostToBatchSummary(summary, share, shareCNY)
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
			applyWalletCostToTimeseriesPoint(point, share, shareCNY)
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
			applyWalletCostToBreakdownItem(item, share, shareCNY)
		}
		for index := range report.ModelBreakdown {
			item := &report.ModelBreakdown[index]
			requests := batchRequests[item.BatchId]
			if requests <= 0 {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, totalBucketRequests, item.RequestCount)
			shareCNY := profitBoardConfiguredUpstreamCostCNY(share, comboPricingMap[item.BatchId])
			applyWalletCostToBreakdownItem(item, share, shareCNY)
		}
	}
	report.Summary.RemoteObservedCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostCNY += appliedTotalCostCNY
	report.Summary.ConfiguredProfitUSD -= appliedTotalCostUSD
	report.Summary.ConfiguredProfitCNY -= appliedTotalCostCNY
	report.Summary.ActualProfitUSD -= appliedTotalCostUSD
}

// applyProfitBoardObservedWalletCostCumulative 按累计窗口按组合请求量加权分摊钱包观测成本。
// 累计总览下快照 SyncedAt 通常只落在少数日期，若按桶分摊会让大多数有请求的组合分不到成本。
// 这里改为：对每个观测点，取其 SyncedAt 时刻仍有效的组合（尊重 CreatedAt），按各组合整窗请求数加权分摊。
func applyProfitBoardObservedWalletCostCumulative(report *ProfitBoardReport, aggregate *profitBoardUpstreamAccountObservedAggregate, comboPricingMap map[string]profitBoardResolvedComboPricing, comboIDSet map[string]ProfitBoardBatchInfo) {
	batchRequests := make(map[string]int, len(comboIDSet))
	for index := range report.BatchSummaries {
		summary := &report.BatchSummaries[index]
		if _, ok := comboIDSet[summary.BatchId]; !ok {
			continue
		}
		batchRequests[summary.BatchId] = summary.RequestCount
	}

	appliedTotalCostUSD := 0.0
	appliedTotalCostCNY := 0.0
	for _, observedPoint := range aggregate.Points {
		if observedPoint.CostUSD <= 0 {
			continue
		}
		activeBatchIDs := make([]string, 0, len(comboIDSet))
		activeTotalRequests := 0
		for comboID, batch := range comboIDSet {
			if batch.CreatedAt > 0 && observedPoint.SyncedAt < batch.CreatedAt {
				continue
			}
			activeBatchIDs = append(activeBatchIDs, comboID)
			activeTotalRequests += batchRequests[comboID]
		}
		if len(activeBatchIDs) == 0 {
			continue
		}

		appliedTotalCostUSD += observedPoint.CostUSD

		if activeTotalRequests <= 0 {
			// 所有活跃组合均无请求，退回按组合数均分，沿用 per-bucket 路径的 fallback 口径。
			activeCount := len(activeBatchIDs)
			shareUSD := observedPoint.CostUSD / float64(activeCount)
			for _, batchID := range activeBatchIDs {
				shareCNY := profitBoardConfiguredUpstreamCostCNY(shareUSD, comboPricingMap[batchID])
				appliedTotalCostCNY += shareCNY
				applyWalletCostCumulativeToReport(report, batchID, shareUSD, shareCNY, batchRequests[batchID])
			}
			continue
		}

		for _, batchID := range activeBatchIDs {
			requests := batchRequests[batchID]
			if requests <= 0 {
				continue
			}
			share := profitBoardFixedAllocationShare(observedPoint.CostUSD, activeTotalRequests, requests)
			if share == 0 {
				continue
			}
			shareCNY := profitBoardConfiguredUpstreamCostCNY(share, comboPricingMap[batchID])
			appliedTotalCostCNY += shareCNY
			applyWalletCostCumulativeToReport(report, batchID, share, shareCNY, requests)
		}
	}

	report.Summary.RemoteObservedCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostUSD += appliedTotalCostUSD
	report.Summary.UpstreamCostCNY += appliedTotalCostCNY
	report.Summary.ConfiguredProfitUSD -= appliedTotalCostUSD
	report.Summary.ConfiguredProfitCNY -= appliedTotalCostCNY
	report.Summary.ActualProfitUSD -= appliedTotalCostUSD
}

func applyWalletCostCumulativeToReport(report *ProfitBoardReport, batchID string, shareUSD float64, shareCNY float64, batchTotalRequests int) {
	for index := range report.BatchSummaries {
		summary := &report.BatchSummaries[index]
		if summary.BatchId != batchID {
			continue
		}
		applyWalletCostToBatchSummary(summary, shareUSD, shareCNY)
		break
	}
	if batchTotalRequests <= 0 {
		return
	}
	for index := range report.ChannelBreakdown {
		item := &report.ChannelBreakdown[index]
		if item.BatchId != batchID || item.RequestCount <= 0 {
			continue
		}
		itemShareUSD := profitBoardFixedAllocationShare(shareUSD, batchTotalRequests, item.RequestCount)
		itemShareCNY := profitBoardFixedAllocationShare(shareCNY, batchTotalRequests, item.RequestCount)
		applyWalletCostToBreakdownItem(item, itemShareUSD, itemShareCNY)
	}
	for index := range report.ModelBreakdown {
		item := &report.ModelBreakdown[index]
		if item.BatchId != batchID || item.RequestCount <= 0 {
			continue
		}
		itemShareUSD := profitBoardFixedAllocationShare(shareUSD, batchTotalRequests, item.RequestCount)
		itemShareCNY := profitBoardFixedAllocationShare(shareCNY, batchTotalRequests, item.RequestCount)
		applyWalletCostToBreakdownItem(item, itemShareUSD, itemShareCNY)
	}
}
