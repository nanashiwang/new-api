package model

func applyProfitBoardComboFixedTotals(report *ProfitBoardReport, comboPricingMap map[string]profitBoardResolvedComboPricing, allocation profitBoardSiteRevenueAllocation, batches []ProfitBoardBatchInfo, startTimestamp int64, endTimestamp int64, granularity string, customIntervalMinutes int) {
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
	totalSiteFixedCNY := 0.0
	totalUpstreamFixed := 0.0
	totalUpstreamFixedCNY := 0.0
	for index := range report.BatchSummaries {
		batchSummary := &report.BatchSummaries[index]
		batchRequestCount[batchSummary.BatchId] = batchSummary.RequestCount
		comboPricing := comboPricingMap[batchSummary.BatchId]
		createdAt := batchCreatedAt[batchSummary.BatchId]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		eligibleBatchRequests := allocation.EligibleBatchRequestCount[batchSummary.BatchId]
		shouldApplySiteFixed := eligibleBatchRequests > 0 || batchSummary.RequestCount == 0
		if shouldApplySiteFixed {
			siteFixedCNY := profitBoardConfiguredSiteRevenueCNY(comboPricing.SiteFixedTotalAmount, comboPricing)
			totalSiteFixed += comboPricing.SiteFixedTotalAmount
			totalSiteFixedCNY += siteFixedCNY
			batchSummary.ConfiguredSiteRevenueUSD += comboPricing.SiteFixedTotalAmount
			batchSummary.ConfiguredSiteRevenueCNY += siteFixedCNY
			batchSummary.ConfiguredProfitUSD += comboPricing.SiteFixedTotalAmount
			batchSummary.ConfiguredProfitCNY += siteFixedCNY
		}
		upstreamFixedCNY := profitBoardConfiguredUpstreamCostCNY(comboPricing.UpstreamFixedTotalAmount, comboPricing)
		totalUpstreamFixed += comboPricing.UpstreamFixedTotalAmount
		totalUpstreamFixedCNY += upstreamFixedCNY
		batchSummary.UpstreamCostUSD += comboPricing.UpstreamFixedTotalAmount
		batchSummary.UpstreamCostCNY += upstreamFixedCNY
		batchSummary.ConfiguredProfitUSD -= comboPricing.UpstreamFixedTotalAmount
		batchSummary.ConfiguredProfitCNY -= upstreamFixedCNY
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
		if allocation.EligibleBatchRequestCount[batch.Id] > 0 || batchRequestCount[batch.Id] == 0 {
			siteFixedCNY := profitBoardConfiguredSiteRevenueCNY(comboPricing.SiteFixedTotalAmount, comboPricing)
			point.ConfiguredSiteRevenueUSD += comboPricing.SiteFixedTotalAmount
			point.ConfiguredSiteRevenueCNY += siteFixedCNY
			point.ConfiguredProfitUSD += comboPricing.SiteFixedTotalAmount
			point.ConfiguredProfitCNY += siteFixedCNY
		}
		upstreamFixedCNY := profitBoardConfiguredUpstreamCostCNY(comboPricing.UpstreamFixedTotalAmount, comboPricing)
		point.UpstreamCostUSD += comboPricing.UpstreamFixedTotalAmount
		point.UpstreamCostCNY += upstreamFixedCNY
		point.ConfiguredProfitUSD -= comboPricing.UpstreamFixedTotalAmount
		point.ConfiguredProfitCNY -= upstreamFixedCNY
		point.ActualProfitUSD -= comboPricing.UpstreamFixedTotalAmount
	}

	for index := range report.ChannelBreakdown {
		item := &report.ChannelBreakdown[index]
		totalRequests := batchRequestCount[item.BatchId]
		eligibleRequests := allocation.EligibleChannelRequestCount[item.BatchId+"|"+item.Key]
		eligibleBatchRequests := allocation.EligibleBatchRequestCount[item.BatchId]
		comboPricing := comboPricingMap[item.BatchId]
		createdAt := batchCreatedAt[item.BatchId]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		siteShare := profitBoardFixedAllocationShare(comboPricing.SiteFixedTotalAmount, eligibleBatchRequests, eligibleRequests)
		upstreamShare := profitBoardFixedAllocationShare(comboPricing.UpstreamFixedTotalAmount, totalRequests, item.RequestCount)
		siteShareCNY := profitBoardConfiguredSiteRevenueCNY(siteShare, comboPricing)
		upstreamShareCNY := profitBoardConfiguredUpstreamCostCNY(upstreamShare, comboPricing)
		item.ConfiguredSiteRevenueUSD += siteShare
		item.ConfiguredSiteRevenueCNY += siteShareCNY
		item.UpstreamCostUSD += upstreamShare
		item.UpstreamCostCNY += upstreamShareCNY
		item.ConfiguredProfitUSD += siteShare - upstreamShare
		item.ConfiguredProfitCNY += siteShareCNY - upstreamShareCNY
		item.ActualProfitUSD -= upstreamShare
	}

	for index := range report.ModelBreakdown {
		item := &report.ModelBreakdown[index]
		totalRequests := batchRequestCount[item.BatchId]
		eligibleRequests := allocation.EligibleModelRequestCount[item.BatchId+"|"+item.Key]
		eligibleBatchRequests := allocation.EligibleBatchRequestCount[item.BatchId]
		comboPricing := comboPricingMap[item.BatchId]
		createdAt := batchCreatedAt[item.BatchId]
		if !profitBoardTimestampInRange(createdAt, startTimestamp, endTimestamp) {
			continue
		}
		siteShare := profitBoardFixedAllocationShare(comboPricing.SiteFixedTotalAmount, eligibleBatchRequests, eligibleRequests)
		upstreamShare := profitBoardFixedAllocationShare(comboPricing.UpstreamFixedTotalAmount, totalRequests, item.RequestCount)
		siteShareCNY := profitBoardConfiguredSiteRevenueCNY(siteShare, comboPricing)
		upstreamShareCNY := profitBoardConfiguredUpstreamCostCNY(upstreamShare, comboPricing)
		item.ConfiguredSiteRevenueUSD += siteShare
		item.ConfiguredSiteRevenueCNY += siteShareCNY
		item.UpstreamCostUSD += upstreamShare
		item.UpstreamCostCNY += upstreamShareCNY
		item.ConfiguredProfitUSD += siteShare - upstreamShare
		item.ConfiguredProfitCNY += siteShareCNY - upstreamShareCNY
		item.ActualProfitUSD -= upstreamShare
	}

	report.Meta.SiteFixedTotalAmount = roundProfitBoardAmount(totalSiteFixed)
	report.Meta.UpstreamFixedTotalAmount = roundProfitBoardAmount(totalUpstreamFixed)
	report.Summary.ConfiguredSiteRevenueUSD += totalSiteFixed
	report.Summary.ConfiguredSiteRevenueCNY += totalSiteFixedCNY
	report.Summary.UpstreamCostUSD += totalUpstreamFixed
	report.Summary.UpstreamCostCNY += totalUpstreamFixedCNY
	report.Summary.ConfiguredProfitUSD += totalSiteFixed
	report.Summary.ConfiguredProfitUSD -= totalUpstreamFixed
	report.Summary.ConfiguredProfitCNY += totalSiteFixedCNY
	report.Summary.ConfiguredProfitCNY -= totalUpstreamFixedCNY
	report.Summary.ActualProfitUSD -= totalUpstreamFixed
}
