package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestAdaptorGetRequestURL_AzureResponsesCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info *relaycommon.RelayInfo
		want string
	}{
		{
			name: "azure openai domain uses v1 responses compact preview",
			info: &relaycommon.RelayInfo{
				RelayMode: relayconstant.RelayModeResponsesCompact,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:    constant.ChannelTypeAzure,
					ChannelBaseUrl: "https://example.openai.azure.com",
				},
			},
			want: "https://example.openai.azure.com/openai/v1/responses/compact?api-version=preview",
		},
		{
			name: "azure cognitive services domain uses responses compact with api version",
			info: &relaycommon.RelayInfo{
				RelayMode: relayconstant.RelayModeResponsesCompact,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:    constant.ChannelTypeAzure,
					ChannelBaseUrl: "https://example.cognitiveservices.azure.com",
					ApiVersion:     "2025-04-01-preview",
				},
			},
			want: "https://example.cognitiveservices.azure.com/openai/responses/compact?api-version=2025-04-01-preview",
		},
		{
			name: "azure custom responses version is respected for compact",
			info: &relaycommon.RelayInfo{
				RelayMode: relayconstant.RelayModeResponsesCompact,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:          constant.ChannelTypeAzure,
					ChannelBaseUrl:       "https://example.openai.azure.com",
					ChannelOtherSettings: dto.ChannelOtherSettings{},
				},
			},
			want: "https://example.openai.azure.com/openai/v1/responses/compact?api-version=preview",
		},
	}

	tests[2].info.ChannelOtherSettings.AzureResponsesVersion = "2025-05-01-preview"
	tests[2].want = "https://example.openai.azure.com/openai/v1/responses/compact?api-version=2025-05-01-preview"

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adaptor := &Adaptor{}
			got, err := adaptor.GetRequestURL(tt.info)
			if err != nil {
				t.Fatalf("GetRequestURL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("GetRequestURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
