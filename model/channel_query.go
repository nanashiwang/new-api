package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

func channelOrderClause(idSort bool) string {
	if idSort {
		return "id desc"
	}
	return "priority desc"
}

func channelBaseURLCol() string {
	if common.UsingPostgreSQL {
		return `"base_url"`
	}
	return "`base_url`"
}

func normalizeChannelSearchGroup(group string) string {
	group = strings.TrimSpace(group)
	if strings.EqualFold(group, "null") {
		return ""
	}
	return group
}

func applyChannelListFilters(query *gorm.DB, statusFilter int, typeFilter int) *gorm.DB {
	switch statusFilter {
	case common.ChannelStatusEnabled:
		query = query.Where("status = ?", common.ChannelStatusEnabled)
	case 0:
		query = query.Where("status <> ?", common.ChannelStatusEnabled)
	}
	if typeFilter >= 0 {
		query = query.Where("type = ?", typeFilter)
	}
	return query
}

func applyChannelKeywordFilter(query *gorm.DB, keyword string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return query
	}
	like := "%" + keyword + "%"
	return query.Where(
		"(id = ? OR name LIKE ? OR "+commonKeyCol+" = ? OR "+channelBaseURLCol()+" LIKE ?)",
		common.String2Int(keyword),
		like,
		keyword,
		like,
	)
}

func applyAbilitySearchFilter(query *gorm.DB, group string, modelKeyword string) *gorm.DB {
	group = normalizeChannelSearchGroup(group)
	modelKeyword = strings.TrimSpace(modelKeyword)
	if group == "" && modelKeyword == "" {
		return query
	}

	subQuery := DB.Model(&Ability{}).
		Select("1").
		Where("abilities.channel_id = channels.id")
	if group != "" {
		subQuery = subQuery.Where("abilities."+commonGroupCol+" = ?", group)
	}
	if modelKeyword != "" {
		subQuery = subQuery.Where("abilities.model LIKE ?", "%"+modelKeyword+"%")
	}
	return query.Where("EXISTS (?)", subQuery)
}

func buildChannelQuery(statusFilter int, typeFilter int, omitKey bool) *gorm.DB {
	query := DB.Model(&Channel{})
	if omitKey {
		query = query.Omit("key")
	}
	return applyChannelListFilters(query, statusFilter, typeFilter)
}

func buildChannelSearchQuery(keyword string, group string, modelKeyword string, statusFilter int, typeFilter int, omitKey bool) *gorm.DB {
	query := buildChannelQuery(statusFilter, typeFilter, omitKey)
	query = applyChannelKeywordFilter(query, keyword)
	return applyAbilitySearchFilter(query, group, modelKeyword)
}

func GetChannelsByTagWithFilters(tag string, idSort bool, selectAll bool, statusFilter int, typeFilter int) ([]*Channel, error) {
	var channels []*Channel
	query := buildChannelQuery(statusFilter, typeFilter, !selectAll).
		Where("tag = ?", tag).
		Order(channelOrderClause(idSort))
	err := query.Find(&channels).Error
	return channels, err
}

func SearchChannelsWithFilters(keyword string, group string, modelKeyword string, idSort bool, statusFilter int, typeFilter int) ([]*Channel, error) {
	var channels []*Channel
	err := buildChannelSearchQuery(keyword, group, modelKeyword, statusFilter, typeFilter, true).
		Order(channelOrderClause(idSort)).
		Find(&channels).Error
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func GetPaginatedTagsWithFilters(offset int, limit int, statusFilter int, typeFilter int) ([]*string, error) {
	var tags []*string
	query := buildChannelQuery(statusFilter, typeFilter, false).
		Select("DISTINCT tag").
		Where("tag IS NOT NULL AND tag != ''")
	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&tags).Error
	return tags, err
}

func SearchTagsWithFilters(keyword string, group string, modelKeyword string, idSort bool, statusFilter int, typeFilter int) ([]*string, error) {
	var tags []*string
	subQuery := buildChannelSearchQuery(keyword, group, modelKeyword, statusFilter, typeFilter, true).
		Select("tag").
		Where("tag IS NOT NULL AND tag != ''").
		Order(channelOrderClause(idSort))
	err := DB.Table("(?) as sub", subQuery).
		Select("DISTINCT tag").
		Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func CountAllTagsWithFilters(statusFilter int, typeFilter int) (int64, error) {
	var total int64
	err := buildChannelQuery(statusFilter, typeFilter, false).
		Where("tag IS NOT NULL AND tag != ''").
		Distinct("tag").
		Count(&total).Error
	return total, err
}

func SearchChannelTypeCounts(keyword string, group string, modelKeyword string, statusFilter int) (map[int64]int64, error) {
	type queryResult struct {
		Type  int64 `gorm:"column:type"`
		Count int64 `gorm:"column:count"`
	}

	var results []queryResult
	err := buildChannelSearchQuery(keyword, group, modelKeyword, statusFilter, -1, false).
		Select("type, count(*) as count").
		Group("type").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	typeCounts := make(map[int64]int64, len(results))
	for _, result := range results {
		typeCounts[result.Type] = result.Count
	}
	return typeCounts, nil
}
