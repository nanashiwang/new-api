package controller

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/thanhpk/randstr"
)

type SubscriptionStripePayRequest struct {
	PlanId           int    `json:"plan_id"`
	PurchaseMode     string `json:"purchase_mode"`
	PurchaseQuantity int    `json:"purchase_quantity"`
	// RenewTargetSubscriptionId 仅在 purchase_mode=renew 时生效。
	RenewTargetSubscriptionId int `json:"renew_target_subscription_id"`
}

func SubscriptionRequestStripePay(c *gin.Context) {
	var req SubscriptionStripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.StripePriceId == "" {
		common.ApiErrorMsg(c, "该套餐未配置 StripePriceId")
		return
	}
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		common.ApiErrorMsg(c, "Stripe 未配置或密钥无效")
		return
	}
	if setting.StripeWebhookSecret == "" {
		common.ApiErrorMsg(c, "Stripe Webhook 未配置")
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}
	purchaseQuantity, err := normalizeSubscriptionPurchaseQuantity(userId, req.PurchaseQuantity, plan)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	// 续费模式支持手动指定目标订阅；未指定时按规则自动命中（唯一或最早到期）。
	purchaseMode, renewTargetSubId, err := resolveSubscriptionPurchaseModeAndTarget(
		userId, plan.Id, req.PurchaseMode, req.RenewTargetSubscriptionId,
	)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := checkSubscriptionOrderLimits(userId, plan, purchaseMode, purchaseQuantity); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	reference := fmt.Sprintf("sub-stripe-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "sub_ref_" + common.Sha1([]byte(reference))

	payLink, err := genStripeSubscriptionLink(referenceId, user.StripeCustomer, user.Email, plan.StripePriceId, purchaseQuantity)
	if err != nil {
		log.Println("获取Stripe Checkout支付链接失败", err)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	totalMoney := plan.PriceAmount * float64(purchaseQuantity)

	order := &model.SubscriptionOrder{
		UserId:                    userId,
		PlanId:                    plan.Id,
		Money:                     totalMoney,
		PurchaseQuantity:          purchaseQuantity,
		TradeNo:                   referenceId,
		PaymentMethod:             PaymentMethodStripe,
		PurchaseMode:              purchaseMode,
		RenewTargetSubscriptionId: renewTargetSubId,
		CreateTime:                time.Now().Unix(),
		Status:                    common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
		},
	})
}

func genStripeSubscriptionLink(referenceId string, customerId string, email string, priceId string, purchaseQuantity int) (string, error) {
	if purchaseQuantity <= 0 {
		purchaseQuantity = 1
	}
	stripe.Key = setting.StripeApiSecret

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(referenceId),
		SuccessURL:        stripe.String(system_setting.ServerAddress + "/console/topup"),
		CancelURL:         stripe.String(system_setting.ServerAddress + "/console/topup"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceId),
				Quantity: stripe.Int64(int64(purchaseQuantity)),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
	}

	if "" == customerId {
		if "" != email {
			params.CustomerEmail = stripe.String(email)
		}
		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
	} else {
		params.Customer = stripe.String(customerId)
	}

	result, err := session.New(params)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
