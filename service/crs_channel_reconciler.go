package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func ReconcileCRSManagedChannels(siteID int, platform string, snapshots []*model.CRSAccountSnapshot, observedAt int64) error {
	return ReconcileCRSManagedChannelsContext(context.Background(), siteID, platform, snapshots, observedAt)
}

func ReconcileCRSManagedChannelsContext(ctx context.Context, siteID int, platform string, snapshots []*model.CRSAccountSnapshot, observedAt int64) error {
	total, healthy, complete := summarizeCRSOpenAIHealth(platform, snapshots)
	if !complete || total == 0 {
		return nil
	}
	channels := make([]*model.Channel, 0)
	if err := model.DB.WithContext(ctx).Find(&channels).Error; err != nil {
		return err
	}
	for _, channel := range channels {
		settings := channel.GetSetting()
		if !settings.CRSAutoManage || settings.CRSSiteID != siteID || settings.CRSPlatform != platform || !channel.GetAutoBan() {
			continue
		}
		if healthy == 0 {
			OpenChannelShortCircuit(channel)
		}
		transition, err := model.ObserveCRSManagedChannelContext(ctx, channel.Id, siteID, platform, observedAt, healthy > 0)
		if err != nil {
			return err
		}
		switch transition {
		case model.CRSChannelTransitionDisabled:
			common.SysLog(fmt.Sprintf("CRS auto-disabled channel #%d: site=%d total=%d healthy=%d", channel.Id, siteID, total, healthy))
		case model.CRSChannelTransitionEnabled:
			common.SysLog(fmt.Sprintf("CRS auto-enabled channel #%d: site=%d total=%d healthy=%d", channel.Id, siteID, total, healthy))
		}
	}
	return nil
}

func summarizeCRSOpenAIHealth(platform string, snapshots []*model.CRSAccountSnapshot) (total, healthy int, complete bool) {
	complete = true
	for _, snapshot := range snapshots {
		if snapshot == nil || snapshot.Platform != platform {
			continue
		}
		total++
		active, schedulable, rateLimited, valid := explicitCRSAccountHealth(snapshot.RawAccount)
		if !valid {
			complete = false
			continue
		}
		if active && schedulable && !rateLimited {
			healthy++
		}
	}
	return total, healthy, complete
}

func explicitCRSAccountHealth(raw string) (active, schedulable, rateLimited, valid bool) {
	account := make(map[string]any)
	if raw == "" || common.UnmarshalJsonStr(raw, &account) != nil {
		return false, false, false, false
	}
	active, activeOK := account["isActive"].(bool)
	schedulable, schedulableOK := account["schedulable"].(bool)
	rateLimit, rateLimitOK := account["rateLimitStatus"].(map[string]any)
	if !rateLimitOK {
		return active, schedulable, false, false
	}
	rateLimited, rateLimitedOK := rateLimit["isRateLimited"].(bool)
	return active, schedulable, rateLimited, activeOK && schedulableOK && rateLimitedOK
}
