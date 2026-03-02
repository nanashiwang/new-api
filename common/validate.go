package common

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validate 全局验证器实例
var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

// fieldNameMap 结构体字段名到中文显示名的映射
var fieldNameMap = map[string]string{
	"Username":    "用户名",
	"Password":    "密码",
	"DisplayName": "显示名称",
	"Email":       "邮箱",
	"Remark":      "备注",
}

// TranslateValidationErrors 将验证器错误信息翻译为中文提示
func TranslateValidationErrors(err error) string {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	var msgs []string
	for _, e := range validationErrors {
		fieldName := e.Field()
		if cn, exists := fieldNameMap[fieldName]; exists {
			fieldName = cn
		}

		switch e.Tag() {
		case "min":
			msgs = append(msgs, fmt.Sprintf("%s长度不能少于%s个字符", fieldName, e.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("%s长度不能超过%s个字符", fieldName, e.Param()))
		case "required":
			msgs = append(msgs, fmt.Sprintf("%s不能为空", fieldName))
		case "email":
			msgs = append(msgs, fmt.Sprintf("%s格式不正确", fieldName))
		default:
			msgs = append(msgs, fmt.Sprintf("%s验证失败", fieldName))
		}
	}
	return strings.Join(msgs, "; ")
}
