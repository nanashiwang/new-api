package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const upstreamAccountSyncInterval = 5 * time.Minute

var upstreamAccountSyncOnce sync.Once

func StartUpstreamAccountSyncTask() {
	upstreamAccountSyncOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		go func() {
			common.SysLog("upstream account sync task started")
			syncEnabledUpstreamAccounts()

			ticker := time.NewTicker(upstreamAccountSyncInterval)
			defer ticker.Stop()
			for range ticker.C {
				syncEnabledUpstreamAccounts()
			}
		}()
	})
}

func syncEnabledUpstreamAccounts() {
	accounts, err := listUpstreamAccounts()
	if err != nil {
		common.SysError("list upstream accounts for sync failed: " + err.Error())
		return
	}
	for _, account := range accounts {
		if !account.Enabled {
			continue
		}
		if _, err := SyncUpstreamAccount(account.Id, false); err != nil {
			common.SysError(fmt.Sprintf("upstream account %d sync failed: %v", account.Id, err))
		}
	}
}
