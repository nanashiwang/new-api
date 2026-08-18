package billing_setting

import "testing"

func TestParseAndValidateBundleRejectsMissingAndUnsafeExpressions(t *testing.T) {
	for _, raw := range []string{
		`{"billing_mode":{"m":"tiered_expr"},"billing_expr":{}}`,
		`{"billing_mode":{"m":"tiered_expr"},"billing_expr":{"m":"1 / c"}}`,
		`{"billing_mode":{"m":"tiered_expr"},"billing_expr":{"m":"p - c * 2"}}`,
	} {
		if _, err := ParseAndValidateBundle(raw); err == nil {
			t.Fatalf("expected invalid bundle to be rejected: %s", raw)
		}
	}
}

func TestParseAndValidateFieldsRejectsMalformedJSON(t *testing.T) {
	if _, err := ParseAndValidateFields(`{"m":"tiered_expr"}`, `{`); err == nil {
		t.Fatal("expected malformed expression JSON to be rejected")
	}
}

func TestReplaceBundlePublishesModeAndExpressionTogether(t *testing.T) {
	saved := TieredBundle{BillingMode: GetBillingModeCopy(), BillingExpr: GetBillingExprCopy()}
	t.Cleanup(func() { ReplaceBundle(saved) })

	bundle, err := ParseAndValidateBundle(
		`{"billing_mode":{"m":"tiered_expr"},"billing_expr":{"m":"tier(\"base\", p * 2 + c * 8)"}}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	ReplaceBundle(bundle)
	if GetBillingMode("m") != BillingModeTieredExpr {
		t.Fatal("tiered mode was not published")
	}
	if expr, ok := GetBillingExpr("m"); !ok || expr == "" {
		t.Fatal("tiered expression was not published")
	}
}
