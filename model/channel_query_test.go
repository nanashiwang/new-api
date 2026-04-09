package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelQueryTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originDB := DB
	originLogDB := LOG_DB
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL

	DB = db
	LOG_DB = db
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()

	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		initCol()
	})

	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))
}

func seedChannelQueryTestData(t *testing.T) {
	t.Helper()

	alphaTag := "alpha"
	betaTag := "beta"
	gammaTag := "gamma"
	priority := int64(0)
	weight := uint(10)

	channels := []*Channel{
		{
			Id:       1,
			Name:     "alpha-openai",
			Key:      "alpha-openai-key",
			Type:     1,
			Status:   common.ChannelStatusEnabled,
			Group:    "default,vip",
			Models:   "gpt-4o,claude-3-5-sonnet",
			Tag:      &alphaTag,
			Priority: &priority,
			Weight:   &weight,
		},
		{
			Id:       2,
			Name:     "alpha-gemini-disabled",
			Key:      "alpha-gemini-key",
			Type:     2,
			Status:   common.ChannelStatusManuallyDisabled,
			Group:    "vip",
			Models:   "gemini-1.5-pro",
			Tag:      &alphaTag,
			Priority: &priority,
			Weight:   &weight,
		},
		{
			Id:       3,
			Name:     "beta-mini",
			Key:      "beta-mini-key",
			Type:     2,
			Status:   common.ChannelStatusEnabled,
			Group:    "vip",
			Models:   "gpt-4o-mini",
			Tag:      &betaTag,
			Priority: &priority,
			Weight:   &weight,
		},
		{
			Id:       4,
			Name:     "gamma-basic",
			Key:      "gamma-basic-key",
			Type:     1,
			Status:   common.ChannelStatusEnabled,
			Group:    "default",
			Models:   "gemini-1.5-flash",
			Tag:      &gammaTag,
			Priority: &priority,
			Weight:   &weight,
		},
	}

	for _, channel := range channels {
		require.NoError(t, channel.Insert())
	}
}

func TestSearchChannelsWithFilters_UsesAbilitiesForGroupAndModel(t *testing.T) {
	setupChannelQueryTestDB(t)
	seedChannelQueryTestData(t)

	channels, err := SearchChannelsWithFilters("", "vip", "gemini", false, -1, -1)
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, 2, channels[0].Id)

	enabledOnly, err := SearchChannelsWithFilters("", "vip", "gemini", false, common.ChannelStatusEnabled, -1)
	require.NoError(t, err)
	require.Empty(t, enabledOnly)
}

func TestTagFiltersRespectStatusAndType(t *testing.T) {
	setupChannelQueryTestDB(t)
	seedChannelQueryTestData(t)

	channels, err := GetChannelsByTagWithFilters("alpha", false, false, common.ChannelStatusEnabled, -1)
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, 1, channels[0].Id)

	tags, err := SearchTagsWithFilters("", "vip", "gpt-4", false, common.ChannelStatusEnabled, 1)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.NotNil(t, tags[0])
	require.Equal(t, "alpha", *tags[0])

	total, err := CountAllTagsWithFilters(common.ChannelStatusEnabled, 1)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)

	pagedTags, err := GetPaginatedTagsWithFilters(0, 10, common.ChannelStatusEnabled, 1)
	require.NoError(t, err)
	require.Len(t, pagedTags, 2)
}

func TestSearchChannelTypeCounts_IgnoresTypeFilterByDesign(t *testing.T) {
	setupChannelQueryTestDB(t)
	seedChannelQueryTestData(t)

	typeCounts, err := SearchChannelTypeCounts("", "vip", "gpt-4", common.ChannelStatusEnabled)
	require.NoError(t, err)
	require.EqualValues(t, 1, typeCounts[1])
	require.EqualValues(t, 1, typeCounts[2])
}
