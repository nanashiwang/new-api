package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupChannelPeriodQuotaTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	originDB := model.DB
	originLogDB := model.LOG_DB
	originSQLite, originMySQL, originPG := common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL
	originNow := nowChannelPeriodQuota
	originDisable := disableChannelPeriodQuota
	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = true, false, false
	nowChannelPeriodQuota = func() time.Time { return time.Date(2024, 1, 2, 12, 0, 0, 0, time.Local) }
	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
		common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = originSQLite, originMySQL, originPG
		nowChannelPeriodQuota = originNow
		disableChannelPeriodQuota = originDisable
	})
	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.ChannelTagPolicy{}, &model.ChannelQuotaUsage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func createQuotaTestChannel(t *testing.T, id int, tag string, policy dto.QuotaPolicy) {
	t.Helper()
	ch := model.Channel{Id: id, Key: fmt.Sprintf("k-%d", id), Name: fmt.Sprintf("c-%d", id), Status: common.ChannelStatusEnabled, Models: "gpt", Group: "default"}
	ch.SetTag(tag)
	if policy.Period != "" || policy.Enabled || policy.QuotaLimit != 0 || policy.CountLimit != 0 {
		ch.SetSetting(dto.ChannelSettings{QuotaPolicy: policy})
	}
	if err := model.DB.Create(&ch).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
}

func TestChannelPeriodQuotaChannelLevelTrigger(t *testing.T) {
	setupChannelPeriodQuotaTestDB(t)
	createQuotaTestChannel(t, 1, "pool", dto.QuotaPolicy{Enabled: true, Period: "day", QuotaLimit: 10})
	var calls atomic.Int32
	disableChannelPeriodQuota = func(scope, scopeKey string, channelId int, periodEnd int64) error {
		calls.Add(1)
		if scope != periodQuotaScopeChannel || scopeKey != "1" || channelId != 1 {
			t.Fatalf("bad disable args %s %s %d", scope, scopeKey, channelId)
		}
		return nil
	}
	RecordChannelPeriodQuota(1, 10, 0)
	if calls.Load() != 1 {
		t.Fatalf("want one disable call, got %d", calls.Load())
	}
}

func TestChannelPeriodQuotaChannelOverridesTag(t *testing.T) {
	setupChannelPeriodQuotaTestDB(t)
	createQuotaTestChannel(t, 1, "pool", dto.QuotaPolicy{Enabled: true, Period: "day", CountLimit: 2})
	if err := model.UpsertTagPolicy("pool", dto.QuotaPolicy{Enabled: true, Period: "day", CountLimit: 1}); err != nil {
		t.Fatal(err)
	}
	RecordChannelPeriodCount(1)
	start, _ := common.CalcPeriodWindow("day", nowChannelPeriodQuota())
	if _, err := model.GetChannelQuotaUsage(periodQuotaScopeTag, "pool", "day", start); err == nil {
		t.Fatal("tag usage should not be written when channel policy is active")
	}
	usage, err := model.GetChannelQuotaUsage(periodQuotaScopeChannel, "1", "day", start)
	if err != nil || usage.UsedCount != 1 {
		t.Fatalf("channel usage err=%v usage=%+v", err, usage)
	}
}

func TestChannelPeriodQuotaTagSharedPoolAggregation(t *testing.T) {
	setupChannelPeriodQuotaTestDB(t)
	createQuotaTestChannel(t, 1, "pool", dto.QuotaPolicy{})
	createQuotaTestChannel(t, 2, "pool", dto.QuotaPolicy{})
	if err := model.UpsertTagPolicy("pool", dto.QuotaPolicy{Enabled: true, Period: "day", CountLimit: 2}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	disableChannelPeriodQuota = func(scope, scopeKey string, channelId int, periodEnd int64) error {
		calls.Add(1)
		if scope != periodQuotaScopeTag || scopeKey != "pool" {
			t.Fatalf("bad tag disable args")
		}
		return nil
	}
	RecordChannelPeriodCount(1)
	RecordChannelPeriodCount(2)
	if calls.Load() != 1 {
		t.Fatalf("want one tag disable call, got %d", calls.Load())
	}
}

func TestRecoverPeriodQuotaUsage(t *testing.T) {
	setupChannelPeriodQuotaTestDB(t)
	createQuotaTestChannel(t, 1, "pool", dto.QuotaPolicy{})
	ch, err := model.GetChannelById(1, true)
	if err != nil {
		t.Fatal(err)
	}
	ch.Status = common.ChannelStatusAutoDisabled
	model.SetPeriodQuotaMeta(ch, periodQuotaScopeChannel, "1", 20)
	if err := ch.SaveWithoutKey(); err != nil {
		t.Fatal(err)
	}
	usage := model.ChannelQuotaUsage{Scope: periodQuotaScopeChannel, ScopeKey: "1", PeriodEnd: 20}
	if !recoverPeriodQuotaUsage(usage) {
		t.Fatal("expected recovery")
	}
	ch, _ = model.GetChannelById(1, true)
	if ch.Status != common.ChannelStatusEnabled || model.HasPeriodQuotaMeta(ch) {
		t.Fatalf("not recovered: status=%d other=%s", ch.Status, ch.OtherInfo)
	}
}

func TestChannelPeriodQuotaConcurrentRecordingSingleDisable(t *testing.T) {
	setupChannelPeriodQuotaTestDB(t)
	createQuotaTestChannel(t, 1, "pool", dto.QuotaPolicy{Enabled: true, Period: "day", CountLimit: 1})
	var calls atomic.Int32
	disableChannelPeriodQuota = func(scope, scopeKey string, channelId int, periodEnd int64) error {
		calls.Add(1)
		return nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RecordChannelPeriodCount(1)
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("want exactly one disable call, got %d", calls.Load())
	}
}
