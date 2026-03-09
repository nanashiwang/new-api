package console_setting

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func mustMarshalAnnouncements(t *testing.T, announcements []map[string]interface{}) string {
	t.Helper()

	payload, err := common.Marshal(announcements)
	require.NoError(t, err)

	return string(payload)
}

func makeAnnouncement(content, extra string) map[string]interface{} {
	return map[string]interface{}{
		"id":          1,
		"content":     content,
		"publishDate": time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"type":        "default",
		"extra":       extra,
	}
}

func TestValidateAnnouncements_AllowsChineseByCharacterCount(t *testing.T) {
	announcements := []map[string]interface{}{
		makeAnnouncement(strings.Repeat("你", 500), strings.Repeat("说", 200)),
	}

	err := validateAnnouncements(mustMarshalAnnouncements(t, announcements))
	require.NoError(t, err)
}

func TestValidateAnnouncements_RejectsChineseContentOverLimit(t *testing.T) {
	announcements := []map[string]interface{}{
		makeAnnouncement(strings.Repeat("你", 501), "备注"),
	}

	err := validateAnnouncements(mustMarshalAnnouncements(t, announcements))
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestValidateAnnouncements_RejectsExtraOverLimitByCharacterCount(t *testing.T) {
	announcements := []map[string]interface{}{
		makeAnnouncement("正常公告", strings.Repeat("注", 201)),
	}

	err := validateAnnouncements(mustMarshalAnnouncements(t, announcements))
	require.Error(t, err)
	require.Contains(t, err.Error(), "200")
}

func TestValidateAnnouncements_AllowsEmojiWithinLimit(t *testing.T) {
	announcements := []map[string]interface{}{
		makeAnnouncement(strings.Repeat("😀", 500), strings.Repeat("✨", 200)),
	}

	err := validateAnnouncements(mustMarshalAnnouncements(t, announcements))
	require.NoError(t, err)
}
