package model

func applyProfitBoardComboFixedTotals(report *ProfitBoardReport, comboPricingMap map[string]profitBoardResolvedComboPricing, batches []ProfitBoardBatchInfo, startTimestamp int64, endTimestamp int64, granularity string, customIntervalMinutes int) {
	if report == nil {
		return
	}
	if report.Meta.FixedAmountAllocationMode == "" {
		report.Meta.FixedAmountAllocationMode = "request_count"
	}
	if report.Meta.FixedTotalAmountScope == "" {
		report.Meta.FixedTotalAmountScope = "created_at_once"
	}
	batchCreatedAt := profitBoardBatchCreatedAtMap(batches)
	batchRequestCount := make(map[string]int, len(report.BatchSummaries))
	totalSiteFixed := 0.0
	totalUpstreamFixed := 0.0
	for index := range report.BatchSummaries {
		batchSummary := &report.BatchSummaries[index]
		batchRequestCount[batchSummary.BatchId] = batchSummary.RequestCount
		comboPricing := comboPricingMap[batchSummary.BatchId]
		createdAt := batchCreatedAt[batchSummary.BatchId]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		totalSiteFixed += comboPricing.SiteFixedTotalAmount
		totalUpstreamFixed += comboPricing.UpstreamFixedTotalAmount
		batchSummary.ConfiguredSiteRevenueUSD += comboPricing.SiteFixedTotalAmount
		batchSummary.UpstreamCostUSD += comboPricing.UpstreamFixedTotalAmount
		batchSummary.ConfiguredProfitUSD += comboPricing.SiteFixedTotalAmount - comboPricing.UpstreamFixedTotalAmount
		batchSummary.ActualProfitUSD -= comboPricing.UpstreamFixedTotalAmount
	}

	for _, batch := range batches {
		comboPricing := comboPricingMap[batch.Id]
		createdAt := batchCreatedAt[batch.Id]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		bucketTimestamp, bucketLabel := buildProfitBoardBucket(createdAt, granularity, customIntervalMinutes)
		point := getOrCreateProfitBoardTimeseriesPoint(report, batch.Id, batch.Name, bucketTimestamp, bucketLabel)
		point.ConfiguredSiteRevenueUSD += comboPricing.SiteFixedTotalAmount
		point.UpstreamCostUSD += comboPricing.UpstreamFixedTotalAmount
		point.ConfiguredProfitUSD += comboPricing.SiteFixedTotalAmount - comboPricing.UpstreamFixedTotalAmount
		point.ActualProfitUSD -= comboPricing.UpstreamFixedTotalAmount
	}

	for index := range report.ChannelBreakdown {
		item := &report.ChannelBreakdown[index]
		totalRequests := batchRequestCount[item.BatchId]
		comboPricing := comboPricingMap[item.BatchId]
		createdAt := batchCreatedAt[item.BatchId]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		siteShare := profitBoardFixedAllocationShare(comboPricing.SiteFixedTotalAmount, totalRequests, item.RequestCount)
		upstreamShare := profitBoardFixedAllocationShare(comboPricing.UpstreamFixedTotalAmount, totalRequests, item.RequestCount)
		item.ConfiguredSiteRevenueUSD += siteShare
		item.UpstreamCostUSD += upstreamShare
		item.ConfiguredProfitUSD += siteShare - upstreamShare
		item.ActualProfitUSD -= upstreamShare
	}

	for index := range report.ModelBreakdown {
		item := &report.ModelBreakdown[index]
		totalRequests := batchRequestCount[item.BatchId]
		comboPricing := comboPricingMap[item.BatchId]
		createdAt := batchCreatedAt[item.BatchId]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		siteShare := profitBoardFixedAllocationShare(comboPricing.SiteFixedTotalAmount, totalRequests, item.RequestCount)
		upstreamShare := profitBoardFixedAllocationShare(comboPricing.UpstreamFixedTotalAmount, totalRequests, item.RequestCount)
		item.ConfiguredSiteRevenueUSD += siteShare
		item.UpstreamCostUSD += upstreamShare
		item.ConfiguredProfitUSD += siteShare - upstreamShare
		item.ActualProfitUSD -= upstreamShare
	}

	report.Meta.SiteFixedTotalAmount = roundProfitBoardAmount(totalSiteFixed)
	report.Meta.UpstreamFixedTotalAmount = roundProfitBoardAmount(totalUpstreamFixed)
	report.Summary.ConfiguredSiteRevenueUSD += totalSiteFixed
	report.Summary.UpstreamCostUSD += totalUpstreamFixed
	report.Summary.ConfiguredProfitUSD += totalSiteFixed - totalUpstreamFixed
	report.Summary.ActualProfitUSD -= totalUpstreamFixed
}
