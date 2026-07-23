package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
)

func TestDispatchCRSObserverSitesBoundsConcurrency(t *testing.T) {
	sites := make([]*model.CRSSite, 8)
	for i := range sites {
		sites[i] = &model.CRSSite{Id: 987650000 + i}
	}
	startedCh := make(chan int, len(sites))
	doneCh := make(chan struct{}, len(sites))
	release := make(chan struct{})
	var active atomic.Int32
	var maxActive atomic.Int32

	started := dispatchCRSObserverSites(sites, func(site *model.CRSSite) error {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			maximum := maxActive.Load()
			if current <= maximum || maxActive.CompareAndSwap(maximum, current) {
				break
			}
		}
		startedCh <- site.Id
		<-release
		doneCh <- struct{}{}
		return nil
	})
	if started != crsObserverMaxConcurrentSites {
		close(release)
		t.Fatalf("started %d sites, want %d", started, crsObserverMaxConcurrentSites)
	}
	for range started {
		waitCRSObserverTestSignal(t, startedCh, "site did not start")
	}
	if got := maxActive.Load(); got != crsObserverMaxConcurrentSites {
		close(release)
		t.Fatalf("max concurrency = %d, want %d", got, crsObserverMaxConcurrentSites)
	}
	close(release)
	for range started {
		waitCRSObserverTestSignal(t, doneCh, "site did not finish")
	}
	waitCRSObserverSlotsReleased(t)
}

func TestDispatchCRSObserverSitesSkipsBusySite(t *testing.T) {
	site := &model.CRSSite{Id: 987650100}
	startedCh := make(chan struct{}, 1)
	doneCh := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	syncSite := func(*model.CRSSite) error {
		calls.Add(1)
		startedCh <- struct{}{}
		<-release
		doneCh <- struct{}{}
		return nil
	}

	if started := dispatchCRSObserverSites([]*model.CRSSite{site}, syncSite); started != 1 {
		t.Fatalf("first dispatch started %d sites, want 1", started)
	}
	waitCRSObserverTestSignal(t, startedCh, "first site sync did not start")
	if started := dispatchCRSObserverSites([]*model.CRSSite{site}, syncSite); started != 0 {
		close(release)
		t.Fatalf("busy dispatch started %d sites, want 0", started)
	}
	if calls.Load() != 1 {
		close(release)
		t.Fatalf("sync called %d times, want 1", calls.Load())
	}
	close(release)
	waitCRSObserverTestSignal(t, doneCh, "site sync did not finish")
	waitCRSObserverSlotsReleased(t)
}

func TestDispatchCRSObserverSitesAdvancesByBatch(t *testing.T) {
	const baseID = 987650120
	sites := make([]*model.CRSSite, 6)
	for i := range sites {
		sites[i] = &model.CRSSite{Id: baseID + i}
	}
	crsObserverDispatchMu.Lock()
	crsObserverDispatchCursor = 0
	crsObserverDispatchMu.Unlock()

	runBatch := func() map[int]bool {
		startedCh := make(chan int, crsObserverMaxConcurrentSites)
		doneCh := make(chan struct{}, crsObserverMaxConcurrentSites)
		release := make(chan struct{})
		started := dispatchCRSObserverSites(sites, func(site *model.CRSSite) error {
			startedCh <- site.Id
			<-release
			doneCh <- struct{}{}
			return nil
		})
		if started != crsObserverMaxConcurrentSites {
			close(release)
			t.Fatalf("started %d sites, want %d", started, crsObserverMaxConcurrentSites)
		}
		seen := make(map[int]bool, started)
		for range started {
			seen[waitCRSObserverTestSignal(t, startedCh, "batch site did not start")] = true
		}
		close(release)
		for range started {
			waitCRSObserverTestSignal(t, doneCh, "batch site did not finish")
		}
		waitCRSObserverSlotsReleased(t)
		return seen
	}

	first := runBatch()
	second := runBatch()
	for _, id := range []int{baseID, baseID + 1, baseID + 2, baseID + 3} {
		if !first[id] {
			t.Fatalf("first batch omitted site %d", id)
		}
	}
	for _, id := range []int{baseID + 4, baseID + 5} {
		if !second[id] {
			t.Fatalf("second batch omitted deferred site %d", id)
		}
	}
}

func TestDispatchCRSObserverSitesIsFairWithOneFreeSlot(t *testing.T) {
	const baseID = 987650140
	sites := make([]*model.CRSSite, 8)
	for i := range sites {
		sites[i] = &model.CRSSite{Id: baseID + i}
	}
	crsObserverDispatchMu.Lock()
	crsObserverDispatchCursor = 0
	crsObserverDispatchMu.Unlock()
	for range crsObserverMaxConcurrentSites - 1 {
		if !tryAcquireCRSObserverSyncSlot() {
			t.Fatal("failed to reserve observer sync slot")
		}
	}
	defer func() {
		for range crsObserverMaxConcurrentSites - 1 {
			releaseCRSObserverSyncSlot()
		}
	}()

	seen := make(map[int]bool, len(sites))
	for range len(sites) {
		startedCh := make(chan int, 1)
		doneCh := make(chan struct{}, 1)
		release := make(chan struct{})
		started := dispatchCRSObserverSites(sites, func(site *model.CRSSite) error {
			startedCh <- site.Id
			<-release
			doneCh <- struct{}{}
			return nil
		})
		if started != 1 {
			close(release)
			t.Fatalf("started %d sites with one free slot, want 1", started)
		}
		seen[waitCRSObserverTestSignal(t, startedCh, "fairness site did not start")] = true
		close(release)
		waitCRSObserverTestSignal(t, doneCh, "fairness site did not finish")
		waitCRSObserverSlotCount(t, crsObserverMaxConcurrentSites-1)
	}
	if len(seen) != len(sites) {
		t.Fatalf("covered %d sites after %d rounds, want all sites", len(seen), len(sites))
	}
}

func TestSyncCRSObserverSiteSharesGlobalSlots(t *testing.T) {
	for range crsObserverMaxConcurrentSites {
		if !tryAcquireCRSObserverSyncSlot() {
			t.Fatal("failed to reserve observer sync slot")
		}
	}
	defer func() {
		for range crsObserverMaxConcurrentSites {
			releaseCRSObserverSyncSlot()
		}
	}()

	err := SyncCRSObserverSite(&model.CRSSite{Id: 987650150})
	if !errors.Is(err, ErrCRSObserverSyncBusy) {
		t.Fatalf("SyncCRSObserverSite error = %v, want sync busy", err)
	}
}

func TestFetchCRSObserverSnapshotsHonorsExpiredDeadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	result, err := fetchCRSObserverSnapshots(
		ctx,
		&model.CRSSite{Id: 987650151},
		"token",
		time.Now().Unix(),
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("fetch error = %v, want deadline exceeded", err)
	}
	if len(result.Snapshots) != 0 {
		t.Fatalf("snapshot count = %d, want 0", len(result.Snapshots))
	}
}

func TestGetCRSObserverSiteLockIsStable(t *testing.T) {
	const siteID = 987650200
	var wg sync.WaitGroup
	locks := make(chan *sync.Mutex, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locks <- getCRSObserverSiteLock(siteID)
		}()
	}
	wg.Wait()
	close(locks)

	var first *sync.Mutex
	for lock := range locks {
		if first == nil {
			first = lock
			continue
		}
		if lock != first {
			t.Fatal("same site received different locks")
		}
	}
}

func waitCRSObserverTestSignal[T any](t *testing.T, ch <-chan T, message string) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal(message)
		var zero T
		return zero
	}
}

func waitCRSObserverSlotsReleased(t *testing.T) {
	waitCRSObserverSlotCount(t, 0)
}

func waitCRSObserverSlotCount(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for len(crsObserverSyncSlots) != want {
		if time.Now().After(deadline) {
			t.Fatalf("observer sync slot count = %d, want %d", len(crsObserverSyncSlots), want)
		}
		time.Sleep(time.Millisecond)
	}
}
