package service

import (
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const crsObserverSyncInterval = 10 * time.Minute

var crsObserverSyncOnce sync.Once

func StartCRSObserverSyncTask() {
	crsObserverSyncOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		go func() {
			common.SysLog("crs observer sync task started")
			runCRSObserverSyncOnce()

			ticker := time.NewTicker(crsObserverSyncInterval)
			defer ticker.Stop()

			for range ticker.C {
				runCRSObserverSyncOnce()
			}
		}()
	})
}

func runCRSObserverSyncOnce() {
	sites, err := model.ListCRSSites()
	if err != nil {
		common.SysError("crs observer sync: list sites failed: " + err.Error())
		return
	}

	for _, site := range sites {
		if site == nil {
			continue
		}
		if err := SyncCRSObserverSite(site); err != nil {
			name := strings.TrimSpace(site.Name)
			if name == "" {
				name = strings.TrimSpace(site.Host)
			}
			common.SysError("crs observer sync failed for " + name + ": " + err.Error())
		}
	}
}
