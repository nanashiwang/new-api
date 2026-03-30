package model

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const profitBoardRemoteObserverSyncInterval = 5 * time.Minute

var profitBoardRemoteObserverSyncOnce sync.Once

func StartProfitBoardRemoteObserverSyncTask() {
	profitBoardRemoteObserverSyncOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		go func() {
			common.SysLog("profit board remote observer sync task started")
			runProfitBoardRemoteObserverSyncOnce()

			ticker := time.NewTicker(profitBoardRemoteObserverSyncInterval)
			defer ticker.Stop()

			for range ticker.C {
				runProfitBoardRemoteObserverSyncOnce()
			}
		}()
	})
}

func runProfitBoardRemoteObserverSyncOnce() {
	records := make([]ProfitBoardConfig, 0)
	if err := DB.Find(&records).Error; err != nil {
		common.SysError("profit board remote observer sync: list configs failed: " + err.Error())
		return
	}

	for _, record := range records {
		payload, err := payloadFromProfitBoardConfigRecord(record)
		if err != nil {
			common.SysError("profit board remote observer sync: parse config failed: " + err.Error())
			continue
		}
		if payload == nil || len(payload.Batches) == 0 || !profitBoardHasEnabledRemoteObserver(payload.ComboConfigs) {
			continue
		}
		if _, err = SyncProfitBoardRemoteObservers(*payload, false); err != nil {
			common.SysError("profit board remote observer sync failed: " + err.Error())
		}
	}
}
