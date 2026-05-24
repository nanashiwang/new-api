package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestCheckUserRegisterConflict(t *testing.T) {
	truncateTables(t)

	users := []User{
		{
			Username: "alice",
			Password: "test-password",
			Email:    "alice@example.com",
			Role:     common.RoleCommonUser,
			Status:   common.UserStatusEnabled,
			AffCode:  "alice-aff",
		},
		{
			Username: "deleted",
			Password: "test-password",
			Email:    "deleted@example.com",
			Role:     common.RoleCommonUser,
			Status:   common.UserStatusEnabled,
			AffCode:  "deleted-aff",
		},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Delete(&users[1]).Error)

	tests := []struct {
		name     string
		username string
		email    string
		want     UserRegisterConflict
	}{
		{
			name:     "username exists",
			username: "alice",
			email:    "new@example.com",
			want:     UserRegisterConflictUsername,
		},
		{
			name:     "email exists",
			username: "new",
			email:    "alice@example.com",
			want:     UserRegisterConflictEmail,
		},
		{
			name:     "both exist",
			username: "alice",
			email:    "alice@example.com",
			want:     UserRegisterConflictBoth,
		},
		{
			name:     "soft deleted username still conflicts",
			username: "deleted",
			email:    "other@example.com",
			want:     UserRegisterConflictUsername,
		},
		{
			name:     "no conflict",
			username: "bob",
			email:    "bob@example.com",
			want:     UserRegisterConflictNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckUserRegisterConflict(tt.username, tt.email)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
