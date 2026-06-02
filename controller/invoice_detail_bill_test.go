package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestBuildInvoiceDetailBillHTMLUsesTransactionProofWording(t *testing.T) {
	html := buildInvoiceDetailBillHTML(&model.InvoiceRequest{
		Id:         7,
		UserId:     29,
		Username:   "zhuhongjun823",
		TotalMoney: 414,
		TotalQuota: 1,
		Title:      "南京信息工程大学",
		TaxNumber:  "12320000466006762K",
		Email:      "csx20020320@126.com",
		Items: []model.InvoiceRequestItem{{
			OrderType:     model.PaymentRecordTypeTopUp,
			OrderId:       23,
			TradeNo:       "2026-04-09",
			PaymentMethod: "alipay",
			Money:         414,
			Amount:        1,
			CompleteTime:  1775702400,
		}},
	})

	for _, want := range []string{
		"曜算平台交易明细证明",
		"用户 zhuhongjun823（用户 ID：29）",
		"于曜算平台存在相关交易记录",
		"支付金额合计人民币 ¥414.00",
		"本《曜算平台交易明细证明》",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("detail bill html missing %q:\n%s", want, html)
		}
	}
	for _, unexpected := range []string{
		"发票申请明细证明",
		"额度合计",
		"发票抬头",
		"南京信息工程大学",
		"12320000466006762K",
		"csx20020320@126.com",
	} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("detail bill html should not contain %q:\n%s", unexpected, html)
		}
	}
}
