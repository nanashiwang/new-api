package relay

import "testing"

func TestExtractHealableParam(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "anthropic temperature deprecated",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"` + "`temperature`" + ` is deprecated for this model."}}`,
			want: "temperature",
		},
		{
			name: "top_k not supported",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"` + "`top_k`" + ` is not supported with thinking."}}`,
			want: "top_k",
		},
		{
			name: "openai style unsupported parameter",
			body: `{"error":{"type":"invalid_request_error","message":"Unsupported parameter: top_p"}}`,
			want: "top_p",
		},
		{
			name: "semantic param never healed",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"` + "`max_tokens`" + ` is not supported."}}`,
			want: "",
		},
		{
			name: "unrelated 400",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 210000 tokens > 200000 maximum"}}`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractHealableParam(tc.body); got != tc.want {
				t.Fatalf("extractHealableParam() = %q, want %q", got, tc.want)
			}
		})
	}
}
