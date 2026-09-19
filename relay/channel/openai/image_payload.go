package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// Each data item with a real payload represents one image. A URL plus base64
// on the same item is one image; distinct mixed-format items are NOT deduplicated
// by taking max(urlCount, base64Count), which would silently undercount them.
func parseOpenAIImagePayloads(body []byte) (created int64, images []dto.ImageData, present bool, err error) {
	var response struct {
		Created int64           `json:"created"`
		Data    json.RawMessage `json:"data"`
	}
	if err = common.Unmarshal(body, &response); err != nil {
		return
	}
	created = response.Created
	data := bytes.TrimSpace(response.Data)
	if len(data) == 0 {
		return
	}
	present = true
	switch data[0] {
	case '[':
		err = common.Unmarshal(data, &images)
	case '{':
		var item dto.ImageData
		err = common.Unmarshal(data, &item)
		images = []dto.ImageData{item}
	default:
		err = fmt.Errorf("invalid image data shape")
	}
	if err != nil {
		return
	}
	filtered := images[:0]
	for _, item := range images {
		if strings.TrimSpace(item.Url) != "" || strings.TrimSpace(item.B64Json) != "" {
			filtered = append(filtered, item)
		}
	}
	images = filtered
	if len(images) == 0 {
		err = fmt.Errorf("upstream returned no image payload")
	}
	return
}
