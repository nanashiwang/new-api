package model

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Ability struct {
	Group     string  `json:"group" gorm:"type:varchar(64);primaryKey;autoIncrement:false"`
	Model     string  `json:"model" gorm:"type:varchar(255);primaryKey;autoIncrement:false"`
	ChannelId int     `json:"channel_id" gorm:"primaryKey;autoIncrement:false;index"`
	Enabled   bool    `json:"enabled"`
	Priority  *int64  `json:"priority" gorm:"bigint;default:0;index"`
	Weight    uint    `json:"weight" gorm:"default:0;index"`
	Tag       *string `json:"tag" gorm:"index"`
}

type AbilityWithChannel struct {
	Ability
	ChannelType int `json:"channel_type"`
}

func GetAllEnableAbilityWithChannels() ([]AbilityWithChannel, error) {
	var abilities []AbilityWithChannel
	err := DB.Table("abilities").
		Select("abilities.*, channels.type as channel_type").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("abilities.enabled = ?", true).
		Scan(&abilities).Error
	return abilities, err
}

func GetGroupEnabledModels(group string) []string {
	var models []string
	// Find distinct models
	DB.Table("abilities").Where(commonGroupCol+" = ? and enabled = ?", group, true).Distinct("model").Pluck("model", &models)
	return models
}

func GetEnabledModels() []string {
	var models []string
	// Find distinct models
	DB.Table("abilities").Where("enabled = ?", true).Distinct("model").Pluck("model", &models)
	return models
}

func GetAllEnableAbilities() []Ability {
	var abilities []Ability
	DB.Find(&abilities, "enabled = ?", true)
	return abilities
}

func getPriority(group string, model string, retry int) (int, error) {

	var priorities []int
	err := DB.Model(&Ability{}).
		Select("DISTINCT(priority)").
		Where(commonGroupCol+" = ? and model = ? and enabled = ?", group, model, true).
		Order("priority DESC").              // 按优先级降序排序
		Pluck("priority", &priorities).Error // Pluck用于将查询的结果直接扫描到一个切片中

	if err != nil {
		// 处理错误
		return 0, err
	}

	if len(priorities) == 0 {
		// 如果没有查询到优先级，则返回错误
		return 0, errors.New("数据库一致性被破坏")
	}

	// 确定要使用的优先级
	var priorityToUse int
	if retry >= len(priorities) {
		// 如果重试次数大于优先级数，则使用最小的优先级
		priorityToUse = priorities[len(priorities)-1]
	} else {
		priorityToUse = priorities[retry]
	}
	return priorityToUse, nil
}

func getChannelQuery(group string, model string, retry int, allowedChannels []int, excludeChannels []int) (*gorm.DB, error) {
	maxPrioritySubQuery := DB.Model(&Ability{}).Select("MAX(priority)").Where(commonGroupCol+" = ? and model = ? and enabled = ?", group, model, true)
	if len(allowedChannels) > 0 {
		maxPrioritySubQuery = maxPrioritySubQuery.Where("channel_id IN ?", allowedChannels)
	}
	if len(excludeChannels) > 0 {
		maxPrioritySubQuery = maxPrioritySubQuery.Where("channel_id NOT IN ?", excludeChannels)
	}
	channelQuery := DB.Where(commonGroupCol+" = ? and model = ? and enabled = ? and priority = (?)", group, model, true, maxPrioritySubQuery)
	if len(allowedChannels) > 0 {
		channelQuery = channelQuery.Where("channel_id IN ?", allowedChannels)
	}
	if len(excludeChannels) > 0 {
		channelQuery = channelQuery.Where("channel_id NOT IN ?", excludeChannels)
	}
	if retry != 0 {
		priority, err := getPriority(group, model, retry)
		if err != nil {
			return nil, err
		} else {
			channelQuery = DB.Where(commonGroupCol+" = ? and model = ? and enabled = ? and priority = ?", group, model, true, priority)
			if len(allowedChannels) > 0 {
				channelQuery = channelQuery.Where("channel_id IN ?", allowedChannels)
			}
			if len(excludeChannels) > 0 {
				channelQuery = channelQuery.Where("channel_id NOT IN ?", excludeChannels)
			}
		}
	}

	return channelQuery, nil
}

func GetChannel(group string, modelName string, retry int, allowedChannels []int, excludeChannels []int, filters ...ChannelFilter) (*Channel, error) {
	abilities, err := getFilteredAbilities(group, modelName, allowedChannels, excludeChannels, filters...)
	normalizedModel := ratio_setting.FormatMatchingModelName(modelName)
	if (err != nil || len(abilities) == 0) && normalizedModel != "" && normalizedModel != modelName {
		abilities, err = getFilteredAbilities(group, normalizedModel, allowedChannels, excludeChannels, filters...)
	}
	if err != nil || len(abilities) == 0 {
		return nil, err
	}

	priorities := make([]int64, 0)
	seenPriorities := make(map[int64]struct{})
	for _, ability := range abilities {
		priority := abilityPriority(ability)
		if _, exists := seenPriorities[priority]; !exists {
			seenPriorities[priority] = struct{}{}
			priorities = append(priorities, priority)
		}
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] > priorities[j] })
	if retry >= len(priorities) {
		retry = len(priorities) - 1
	}
	targetPriority := priorities[retry]
	targetAbilities := make([]Ability, 0, len(abilities))
	for _, ability := range abilities {
		if abilityPriority(ability) == targetPriority {
			targetAbilities = append(targetAbilities, ability)
		}
	}

	weightSum := 0
	for _, ability := range targetAbilities {
		weightSum += int(ability.Weight) + 10
	}
	weight := common.GetRandomInt(weightSum)
	channelID := 0
	for _, ability := range targetAbilities {
		weight -= int(ability.Weight) + 10
		if weight <= 0 {
			channelID = ability.ChannelId
			break
		}
	}
	if channelID == 0 {
		return nil, errors.New("channel not found")
	}
	channel := &Channel{}
	if err := DB.First(channel, "id = ?", channelID).Error; err != nil {
		return nil, err
	}
	return channel, nil
}

// GetSatisfiedChannels 返回当前重试层级下、同一目标优先级的全部候选渠道。
// 它与 GetChannel 使用相同的模型归一化、白名单、排除列表和过滤器语义，
// 但不做随机选择，供上层按实时容量压力进行排序和原子准入。
func GetSatisfiedChannels(group string, modelName string, retry int, allowedChannels []int, excludeChannels []int, filters ...ChannelFilter) ([]*Channel, error) {
	abilities, err := getFilteredAbilities(group, modelName, allowedChannels, excludeChannels)
	normalizedModel := ratio_setting.FormatMatchingModelName(modelName)
	if (err != nil || len(abilities) == 0) && normalizedModel != "" && normalizedModel != modelName {
		abilities, err = getFilteredAbilities(group, normalizedModel, allowedChannels, excludeChannels)
	}
	if err != nil || len(abilities) == 0 {
		return nil, err
	}

	channelIDs := make([]int, 0, len(abilities))
	seenChannelIDs := make(map[int]struct{}, len(abilities))
	for _, ability := range abilities {
		if _, exists := seenChannelIDs[ability.ChannelId]; exists {
			continue
		}
		seenChannelIDs[ability.ChannelId] = struct{}{}
		channelIDs = append(channelIDs, ability.ChannelId)
	}
	var loadedChannels []*Channel
	if err := DB.Where("id IN ?", channelIDs).Find(&loadedChannels).Error; err != nil {
		return nil, err
	}
	channelByID := make(map[int]*Channel, len(loadedChannels))
	for _, channel := range loadedChannels {
		channelByID[channel.Id] = channel
	}

	eligibleAbilities := make([]Ability, 0, len(abilities))
	for _, ability := range abilities {
		channel, exists := channelByID[ability.ChannelId]
		if !exists {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", ability.ChannelId)
		}
		pass := true
		for _, filter := range filters {
			if !filter(channel) {
				pass = false
				break
			}
		}
		if pass {
			eligibleAbilities = append(eligibleAbilities, ability)
		}
	}
	if len(eligibleAbilities) == 0 {
		return nil, nil
	}

	priorities := make([]int64, 0)
	seenPriorities := make(map[int64]struct{})
	for _, ability := range eligibleAbilities {
		priority := abilityPriority(ability)
		if _, exists := seenPriorities[priority]; exists {
			continue
		}
		seenPriorities[priority] = struct{}{}
		priorities = append(priorities, priority)
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] > priorities[j] })
	if retry >= len(priorities) {
		retry = len(priorities) - 1
	}
	targetPriority := priorities[retry]

	channels := make([]*Channel, 0, len(eligibleAbilities))
	selectedChannelIDs := make(map[int]struct{}, len(eligibleAbilities))
	for _, ability := range eligibleAbilities {
		if abilityPriority(ability) != targetPriority {
			continue
		}
		if _, exists := selectedChannelIDs[ability.ChannelId]; exists {
			continue
		}
		selectedChannelIDs[ability.ChannelId] = struct{}{}
		channels = append(channels, channelByID[ability.ChannelId])
	}
	return channels, nil
}

func abilityPriority(ability Ability) int64 {
	if ability.Priority == nil {
		return 0
	}
	return *ability.Priority
}

func getFilteredAbilities(group, modelName string, allowedChannels, excludeChannels []int, filters ...ChannelFilter) ([]Ability, error) {
	query := DB.Where(commonGroupCol+" = ? and model = ? and enabled = ?", group, modelName, true)
	if len(allowedChannels) > 0 {
		query = query.Where("channel_id IN ?", allowedChannels)
	}
	if len(excludeChannels) > 0 {
		query = query.Where("channel_id NOT IN ?", excludeChannels)
	}
	abilities := make([]Ability, 0)
	if err := query.Order("priority DESC, weight DESC").Find(&abilities).Error; err != nil {
		return nil, err
	}
	if len(filters) == 0 {
		return abilities, nil
	}
	filtered := make([]Ability, 0, len(abilities))
	for _, ability := range abilities {
		channel := &Channel{}
		if err := DB.First(channel, "id = ?", ability.ChannelId).Error; err != nil {
			return nil, err
		}
		pass := true
		for _, filter := range filters {
			if !filter(channel) {
				pass = false
				break
			}
		}
		if pass {
			filtered = append(filtered, ability)
		}
	}
	return filtered, nil
}

func (channel *Channel) AddAbilities(tx *gorm.DB) error {
	useDB := DB
	if tx != nil {
		useDB = tx
	}
	return insertAbilities(useDB, buildAbilitiesForChannel(channel))
}

func (channel *Channel) DeleteAbilities() error {
	return DB.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error
}

func buildAbilitiesForChannel(channel *Channel) []Ability {
	models := channel.GetModels()
	groups := channel.GetGroups()
	abilitySet := make(map[string]struct{}, len(models)*len(groups))
	abilities := make([]Ability, 0, len(models)*len(groups))
	for _, model := range models {
		if model == "" {
			continue
		}
		for _, group := range groups {
			if group == "" {
				continue
			}
			key := group + "|" + model
			if _, exists := abilitySet[key]; exists {
				continue
			}
			abilitySet[key] = struct{}{}
			abilities = append(abilities, Ability{
				Group:     group,
				Model:     model,
				ChannelId: channel.Id,
				Enabled:   channel.Status == common.ChannelStatusEnabled,
				Priority:  channel.Priority,
				Weight:    uint(channel.GetWeight()),
				Tag:       channel.Tag,
			})
		}
	}
	return abilities
}

func buildAbilitiesForChannels(channels []*Channel) []Ability {
	total := 0
	for _, channel := range channels {
		total += len(channel.GetModels()) * len(channel.GetGroups())
	}
	abilities := make([]Ability, 0, total)
	for _, channel := range channels {
		abilities = append(abilities, buildAbilitiesForChannel(channel)...)
	}
	return abilities
}

func insertAbilities(useDB *gorm.DB, abilities []Ability) error {
	if len(abilities) == 0 {
		return nil
	}
	return useDB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(abilities, 100).Error
}

func replaceAbilitiesForChannels(useDB *gorm.DB, channels []*Channel) error {
	if len(channels) == 0 {
		return nil
	}
	ids := make([]int, 0, len(channels))
	for _, channel := range channels {
		ids = append(ids, channel.Id)
	}
	if err := useDB.Where("channel_id IN ?", ids).Delete(&Ability{}).Error; err != nil {
		return err
	}
	return insertAbilities(useDB, buildAbilitiesForChannels(channels))
}

// UpdateAbilities updates abilities of this channel.
// Make sure the channel is completed before calling this function.
func (channel *Channel) UpdateAbilities(tx *gorm.DB) error {
	isNewTx := false
	// 如果没有传入事务，创建新的事务
	if tx == nil {
		tx = DB.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		isNewTx = true
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
	}

	err := replaceAbilitiesForChannels(tx, []*Channel{channel})
	if err != nil {
		if isNewTx {
			tx.Rollback()
		}
		return err
	}

	// 如果是新创建的事务，需要提交
	if isNewTx {
		return tx.Commit().Error
	}

	return nil
}

func UpdateAbilityStatus(channelId int, status bool) error {
	return DB.Model(&Ability{}).Where("channel_id = ?", channelId).Select("enabled").Update("enabled", status).Error
}

func UpdateAbilityStatusByTag(tag string, status bool) error {
	return DB.Model(&Ability{}).Where("tag = ?", tag).Select("enabled").Update("enabled", status).Error
}

func UpdateAbilityByTag(tag string, newTag *string, priority *int64, weight *uint) error {
	ability := Ability{}
	if newTag != nil {
		ability.Tag = newTag
	}
	if priority != nil {
		ability.Priority = priority
	}
	if weight != nil {
		ability.Weight = *weight
	}
	return DB.Model(&Ability{}).Where("tag = ?", tag).Updates(ability).Error
}

var fixLock = sync.Mutex{}

func FixAbility() (int, int, error) {
	lock := fixLock.TryLock()
	if !lock {
		return 0, 0, errors.New("已经有一个修复任务在运行中，请稍后再试")
	}
	defer fixLock.Unlock()

	// truncate abilities table
	if common.UsingSQLite {
		err := DB.Exec("DELETE FROM abilities").Error
		if err != nil {
			common.SysLog(fmt.Sprintf("Delete abilities failed: %s", err.Error()))
			return 0, 0, err
		}
	} else {
		err := DB.Exec("TRUNCATE TABLE abilities").Error
		if err != nil {
			common.SysLog(fmt.Sprintf("Truncate abilities failed: %s", err.Error()))
			return 0, 0, err
		}
	}
	var channels []*Channel
	// Find all channels
	err := DB.Model(&Channel{}).Find(&channels).Error
	if err != nil {
		return 0, 0, err
	}
	if len(channels) == 0 {
		return 0, 0, nil
	}
	successCount := 0
	failCount := 0
	for _, chunk := range lo.Chunk(channels, 50) {
		ids := lo.Map(chunk, func(c *Channel, _ int) int { return c.Id })
		// Delete all abilities of this channel
		err = DB.Where("channel_id IN ?", ids).Delete(&Ability{}).Error
		if err != nil {
			common.SysLog(fmt.Sprintf("Delete abilities failed: %s", err.Error()))
			failCount += len(chunk)
			continue
		}
		// Then add new abilities
		for _, channel := range chunk {
			err = channel.AddAbilities(nil)
			if err != nil {
				common.SysLog(fmt.Sprintf("Add abilities for channel %d failed: %s", channel.Id, err.Error()))
				failCount++
			} else {
				successCount++
			}
		}
	}
	InitChannelCache()
	return successCount, failCount, nil
}
