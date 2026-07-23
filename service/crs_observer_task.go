package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const crsObserverSyncInterval = time.Minute

var crsObserverSyncOnce sync.Once
var crsObserverDispatchMu sync.Mutex
var crsObserverDispatchCursor int

type crsObserverDispatchResult int

const (
	crsObserverDispatchStarted crsObserverDispatchResult = iota
	crsObserverDispatchBusy
	crsObserverDispatchFull
)

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
	dispatchCRSObserverSites(sites, syncCRSObserverSiteLocked)
}

func dispatchCRSObserverSites(sites []*model.CRSSite, syncSite func(*model.CRSSite) error) int {
	if len(sites) == 0 {
		return 0
	}
	crsObserverDispatchMu.Lock()
	defer crsObserverDispatchMu.Unlock()
	start := crsObserverDispatchCursor % len(sites)
	next := (start + 1) % len(sites)
	started := 0
dispatchLoop:
	for offset := range len(sites) {
		index := (start + offset) % len(sites)
		site := sites[index]
		if site == nil {
			continue
		}
		switch tryStartCRSObserverSiteSync(site, syncSite) {
		case crsObserverDispatchStarted:
			started++
			next = (index + 1) % len(sites)
		case crsObserverDispatchFull:
			break dispatchLoop
		}
	}
	crsObserverDispatchCursor = next
	return started
}

func tryStartCRSObserverSiteSync(site *model.CRSSite, syncSite func(*model.CRSSite) error) crsObserverDispatchResult {
	if !tryAcquireCRSObserverSyncSlot() {
		return crsObserverDispatchFull
	}

	lock := getCRSObserverSiteLock(site.Id)
	if !lock.TryLock() {
		releaseCRSObserverSyncSlot()
		return crsObserverDispatchBusy
	}

	go func() {
		defer func() {
			lock.Unlock()
			releaseCRSObserverSyncSlot()
			if recovered := recover(); recovered != nil {
				logCRSObserverSyncError(site, fmt.Errorf("panic: %v", recovered))
			}
		}()
		if err := syncSite(site); err != nil {
			logCRSObserverSyncError(site, err)
		}
	}()
	return crsObserverDispatchStarted
}

func logCRSObserverSyncError(site *model.CRSSite, err error) {
	name := strings.TrimSpace(site.Name)
	if name == "" {
		name = strings.TrimSpace(site.Host)
	}
	common.SysError("crs observer sync failed for " + name + ": " + err.Error())
}
