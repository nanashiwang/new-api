package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type tagQuotaPolicyRequest struct {
	Tag         string          `json:"tag"`
	QuotaPolicy dto.QuotaPolicy `json:"quota_policy"`
}

func GetChannelQuotaUsage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	usage, policy, found, err := service.GetCurrentChannelPeriodQuotaUsage(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"policy": policy, "found": found, "usage": usage})
}

func GetChannelTagQuotaPolicy(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	if tag == "" {
		common.ApiError(c, errors.New("tag cannot be empty"))
		return
	}
	policy, found, err := model.GetTagPolicy(tag)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"tag": tag, "found": found, "quota_policy": policy})
}

func PutChannelTagQuotaPolicy(c *gin.Context) {
	var req tagQuotaPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	req.Tag = strings.TrimSpace(req.Tag)
	if req.Tag == "" {
		common.ApiError(c, errors.New("tag cannot be empty"))
		return
	}
	if err := model.UpsertTagPolicy(req.Tag, req.QuotaPolicy); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteChannelTagQuotaPolicy(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	if tag == "" {
		common.ApiError(c, errors.New("tag cannot be empty"))
		return
	}
	if err := model.DeleteTagPolicy(tag); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetChannelTagQuotaUsage(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	if tag == "" {
		common.ApiError(c, errors.New("tag cannot be empty"))
		return
	}
	usage, policy, found, err := service.GetCurrentTagPeriodQuotaUsage(tag)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"tag": tag, "policy": policy, "found": found, "usage": usage})
}
