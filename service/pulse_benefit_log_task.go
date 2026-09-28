package service

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var pulseBenefitLogOnce sync.Once

// StartPulseBenefitLogTask repairs interrupted deliveries and backfills old
// reward audits even if granting new rewards has since been disabled.
func StartPulseBenefitLogTask() {
	pulseBenefitLogOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				if err := model.SyncPulseBenefitLogs(""); err != nil {
					common.SysError("failed to sync Pulse benefit usage logs: " + err.Error())
				}
				<-ticker.C
			}
		}()
	})
}
