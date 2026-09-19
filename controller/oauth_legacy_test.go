package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGitHubLegacyLoginCannotClaimExistingAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldDB, oldRegister := model.DB, common.RegisterEnabled
	model.DB, common.RegisterEnabled = db, false
	t.Cleanup(func() { model.DB, common.RegisterEnabled = oldDB, oldRegister; _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}))
	owner := model.User{Username: "original-owner", GitHubId: "recycled-name"}
	require.NoError(t, db.Create(&owner).Error)
	provider := &oauth.GitHubProvider{}
	user, err := findOrCreateOAuthUser(nil, provider, &oauth.OAuthUser{
		ProviderUserID: "222", Extra: map[string]any{"legacy_id": "recycled-name"},
	}, nil)
	require.Nil(t, user)
	require.IsType(t, &OAuthLegacyBindingNotConfirmedError{}, err)
	var stored model.User
	require.NoError(t, db.First(&stored, owner.Id).Error)
	require.Equal(t, "recycled-name", stored.GitHubId)

	// Numeric bindings never match somebody else's all-digit login name.
	require.NoError(t, db.Model(&owner).Update("github_id", "111").Error)
	user, err = findOrCreateOAuthUser(nil, provider, &oauth.OAuthUser{
		ProviderUserID: "222", Extra: map[string]any{"legacy_id": "111"},
	}, nil)
	require.Nil(t, user)
	require.IsType(t, &OAuthRegistrationDisabledError{}, err)

	// The immutable ID keeps working even after the owner's login name changes.
	user, err = findOrCreateOAuthUser(nil, provider, &oauth.OAuthUser{
		ProviderUserID: "111", Extra: map[string]any{"legacy_id": "new-name"},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, owner.Id, user.Id)
}
