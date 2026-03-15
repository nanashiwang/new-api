package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type tokenTestRequest struct {
	Model string `json:"model"`
}

func GetTokenModels(c *gin.Context) {
	userId := c.GetInt("id")
	tokenID := strings.TrimSpace(c.Query("token_id"))
	requestedGroup := strings.TrimSpace(c.Query("group"))
	detail, _ := strconv.ParseBool(c.Query("detail"))

	var (
		models []string
		token  *model.Token
		err    error
	)

	if tokenID != "" {
		id := common.String2Int(tokenID)
		if id <= 0 {
			common.ApiError(c, fmt.Errorf("token_id 无效"))
			return
		}
		var tokenErr error
		token, tokenErr = model.GetTokenByIds(id, userId)
		if tokenErr != nil {
			common.ApiError(c, tokenErr)
			return
		}
		models, err = resolveTokenAllowedModels(userId, token)
	} else {
		models, err = resolveRequestedTokenModels(userId, requestedGroup)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if detail {
		modelOptions, optionErr := buildTokenModelOptions(userId, token, requestedGroup)
		if optionErr != nil {
			common.ApiError(c, optionErr)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data":    modelOptions,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
}

func TestToken(c *gin.Context) {
	userId := c.GetInt("id")
	id := common.String2Int(c.Param("id"))
	if id <= 0 {
		common.ApiError(c, fmt.Errorf("id 无效"))
		return
	}

	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	req := tokenTestRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		common.ApiError(c, fmt.Errorf("model 不能为空"))
		return
	}
	isAllowed := false
	matchName := ratio_setting.FormatMatchingModelName(req.Model)
	modelOptions, err := buildTokenModelOptions(userId, token, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, modelOption := range modelOptions {
		if modelOption.Name == req.Model || ratio_setting.FormatMatchingModelName(modelOption.Name) == matchName {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该令牌当前分组不可测试此模型",
			"time":    0.0,
		})
		return
	}
	consumedTime, message, runErr := runTokenRelayTest(c, token, req.Model, "")
	if runErr != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": message,
			"time":    consumedTime,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"time":    consumedTime,
	})
}
