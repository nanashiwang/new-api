package model

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	DB = db
	LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&Task{}, &User{}, &Token{}, &Log{}, &Channel{}, &UserOAuthBinding{}, &TwoFA{}, &TwoFABackupCode{}, &PasskeyCredential{}); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}

func truncateTables(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		for _, entity := range []any{
			&Task{},
			&User{},
			&Token{},
			&Log{},
			&Channel{},
			&UserOAuthBinding{},
			&TwoFA{},
			&TwoFABackupCode{},
			&PasskeyCredential{},
		} {
			DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(entity)
		}
	})
}

func insertTask(t *testing.T, task *Task) {
	t.Helper()
	task.CreatedAt = time.Now().Unix()
	task.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(task).Error)
}

func TestHardDeleteUserByIdDeletesOAuthBindings(t *testing.T) {
	truncateTables(t)

	user := &User{Username: "oauth-hard-delete", Password: "password"}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, CreateUserOAuthBinding(&UserOAuthBinding{
		UserId:         user.Id,
		ProviderId:     1001,
		ProviderUserId: "provider-user-1",
	}))

	require.NoError(t, HardDeleteUserById(user.Id))

	var bindingCount int64
	require.NoError(t, DB.Model(&UserOAuthBinding{}).Where("user_id = ?", user.Id).Count(&bindingCount).Error)
	assert.Equal(t, int64(0), bindingCount)

	var userCount int64
	require.NoError(t, DB.Unscoped().Model(&User{}).Where("id = ?", user.Id).Count(&userCount).Error)
	assert.Equal(t, int64(0), userCount)
}

func TestUserHardDeleteDeletesOAuthBindings(t *testing.T) {
	truncateTables(t)

	user := &User{Username: "oauth-hard-delete-method", Password: "password"}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, CreateUserOAuthBinding(&UserOAuthBinding{
		UserId:         user.Id,
		ProviderId:     1002,
		ProviderUserId: "provider-user-2",
	}))

	require.NoError(t, user.HardDelete())

	var bindingCount int64
	require.NoError(t, DB.Model(&UserOAuthBinding{}).Where("user_id = ?", user.Id).Count(&bindingCount).Error)
	assert.Equal(t, int64(0), bindingCount)
}

func TestHardDeleteUserPurgesAuthenticationData(t *testing.T) {
	truncateTables(t)

	user := &User{Username: "auth-hard-delete", Password: "password"}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(&Token{UserId: user.Id, Name: "auth-token", Key: "auth-token-key"}).Error)
	require.NoError(t, DB.Create(&TwoFA{UserId: user.Id, Secret: "totp-secret", IsEnabled: true}).Error)
	codeHash, err := common.HashBackupCode("ABCD-EFGH")
	require.NoError(t, err)
	require.NoError(t, DB.Create(&TwoFABackupCode{UserId: user.Id, CodeHash: codeHash}).Error)
	require.NoError(t, DB.Create(&PasskeyCredential{
		UserID:          user.Id,
		CredentialID:    "credential-id",
		PublicKey:       "public-key",
		AttestationType: "none",
	}).Error)
	require.NoError(t, CreateUserOAuthBinding(&UserOAuthBinding{
		UserId:         user.Id,
		ProviderId:     1003,
		ProviderUserId: "provider-user-3",
	}))

	require.NoError(t, HardDeleteUserById(user.Id))

	for _, tc := range []struct {
		name  string
		model any
		where string
	}{
		{name: "user", model: &User{}, where: "id = ?"},
		{name: "token", model: &Token{}, where: "user_id = ?"},
		{name: "twofa", model: &TwoFA{}, where: "user_id = ?"},
		{name: "twofa_backup_code", model: &TwoFABackupCode{}, where: "user_id = ?"},
		{name: "passkey", model: &PasskeyCredential{}, where: "user_id = ?"},
		{name: "oauth_binding", model: &UserOAuthBinding{}, where: "user_id = ?"},
	} {
		var count int64
		require.NoError(t, DB.Unscoped().Model(tc.model).Where(tc.where, user.Id).Count(&count).Error, tc.name)
		assert.Equal(t, int64(0), count, tc.name)
	}
}

func TestUserAccessTokenOmittedFromJSON(t *testing.T) {
	accessToken := "secret-access-token"
	user := User{Id: 1, Username: "hidden-token", AccessToken: &accessToken}

	data, err := common.Marshal(user)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "access_token")
	assert.NotContains(t, string(data), accessToken)
}

func TestValidateBackupCodeCanOnlyBeUsedOnce(t *testing.T) {
	truncateTables(t)

	user := &User{Username: "twofa-backup", Password: "password"}
	require.NoError(t, DB.Create(user).Error)
	codeHash, err := common.HashBackupCode("ABCD-EFGH")
	require.NoError(t, err)
	require.NoError(t, DB.Create(&TwoFABackupCode{UserId: user.Id, CodeHash: codeHash}).Error)

	ok, err := ValidateBackupCode(user.Id, "abcd-efgh")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = ValidateBackupCode(user.Id, "ABCD-EFGH")
	require.NoError(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Snapshot / Equal — pure logic tests (no DB)
// ---------------------------------------------------------------------------

func TestSnapshotEqual_Same(t *testing.T) {
	s := taskSnapshot{
		Status:     TaskStatusInProgress,
		Progress:   "50%",
		StartTime:  1000,
		FinishTime: 0,
		FailReason: "",
		ResultURL:  "",
		Data:       json.RawMessage(`{"key":"value"}`),
	}
	assert.True(t, s.Equal(s))
}

func TestSnapshotEqual_DifferentStatus(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage(`{}`)}
	b := taskSnapshot{Status: TaskStatusSuccess, Data: json.RawMessage(`{}`)}
	assert.False(t, a.Equal(b))
}

func TestSnapshotEqual_DifferentProgress(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Progress: "30%", Data: json.RawMessage(`{}`)}
	b := taskSnapshot{Status: TaskStatusInProgress, Progress: "60%", Data: json.RawMessage(`{}`)}
	assert.False(t, a.Equal(b))
}

func TestSnapshotEqual_DifferentData(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage(`{"a":1}`)}
	b := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage(`{"a":2}`)}
	assert.False(t, a.Equal(b))
}

func TestSnapshotEqual_NilVsEmpty(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Data: nil}
	b := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage{}}
	// bytes.Equal(nil, []byte{}) == true
	assert.True(t, a.Equal(b))
}

func TestInitTaskRetainsRequestIDForAsyncBillingCorrelation(t *testing.T) {
	task := InitTask(constant.TaskPlatformSuno, &relaycommon.RelayInfo{
		UserId:      7,
		RequestId:   "request-async-1",
		ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 1},
	})
	require.Equal(t, "request-async-1", task.PrivateData.RequestId)
}

func TestSnapshot_Roundtrip(t *testing.T) {
	task := &Task{
		Status:     TaskStatusInProgress,
		Progress:   "42%",
		StartTime:  1234,
		FinishTime: 5678,
		FailReason: "timeout",
		PrivateData: TaskPrivateData{
			ResultURL: "https://example.com/result.mp4",
		},
		Data: json.RawMessage(`{"model":"test-model"}`),
	}
	snap := task.Snapshot()
	assert.Equal(t, task.Status, snap.Status)
	assert.Equal(t, task.Progress, snap.Progress)
	assert.Equal(t, task.StartTime, snap.StartTime)
	assert.Equal(t, task.FinishTime, snap.FinishTime)
	assert.Equal(t, task.FailReason, snap.FailReason)
	assert.Equal(t, task.PrivateData.ResultURL, snap.ResultURL)
	assert.JSONEq(t, string(task.Data), string(snap.Data))
}

// ---------------------------------------------------------------------------
// UpdateWithStatus CAS — DB integration tests
// ---------------------------------------------------------------------------

func TestUpdateWithStatus_Win(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID:   "task_cas_win",
		Status:   TaskStatusInProgress,
		Progress: "50%",
		Data:     json.RawMessage(`{}`),
	}
	insertTask(t, task)

	task.Status = TaskStatusSuccess
	task.Progress = "100%"
	won, err := task.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	assert.True(t, won)

	var reloaded Task
	require.NoError(t, DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, TaskStatusSuccess, reloaded.Status)
	assert.Equal(t, "100%", reloaded.Progress)
}

func TestUpdateWithStatus_Lose(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID: "task_cas_lose",
		Status: TaskStatusFailure,
		Data:   json.RawMessage(`{}`),
	}
	insertTask(t, task)

	task.Status = TaskStatusSuccess
	won, err := task.UpdateWithStatus(TaskStatusInProgress) // wrong fromStatus
	require.NoError(t, err)
	assert.False(t, won)

	var reloaded Task
	require.NoError(t, DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, TaskStatusFailure, reloaded.Status) // unchanged
}

func TestUpdateWithStatus_ConcurrentWinner(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID: "task_cas_race",
		Status: TaskStatusInProgress,
		Quota:  1000,
		Data:   json.RawMessage(`{}`),
	}
	insertTask(t, task)

	const goroutines = 5
	wins := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			t := &Task{}
			*t = Task{
				ID:       task.ID,
				TaskID:   task.TaskID,
				Status:   TaskStatusSuccess,
				Progress: "100%",
				Quota:    task.Quota,
				Data:     json.RawMessage(`{}`),
			}
			t.CreatedAt = task.CreatedAt
			t.UpdatedAt = time.Now().Unix()
			won, err := t.UpdateWithStatus(TaskStatusInProgress)
			if err == nil {
				wins[idx] = won
			}
		}(i)
	}
	wg.Wait()

	winCount := 0
	for _, w := range wins {
		if w {
			winCount++
		}
	}
	assert.Equal(t, 1, winCount, "exactly one goroutine should win the CAS")
}
