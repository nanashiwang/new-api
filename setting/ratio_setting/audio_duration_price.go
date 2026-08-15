package ratio_setting

import (
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// AudioDurationPrice stores input-audio prices in USD per hour.
// Duration billing is intentionally separate from token audio ratios because
// providers such as MiMo bill ASR by elapsed audio time rather than tokens.
var defaultAudioDurationPrice = map[string]float64{
	"mimo-v2.5-asr": 0.074,
	"MiMo-V2.5-ASR": 0.074,
}

var audioDurationPriceMap = types.NewRWMap[string, float64]()

func AudioDurationPrice2JSONString() string {
	return audioDurationPriceMap.MarshalJSONString()
}

func parseAudioDurationPrices(jsonStr string) (map[string]float64, error) {
	var prices map[string]float64
	if err := common.UnmarshalJsonStr(jsonStr, &prices); err != nil {
		return nil, err
	}
	for modelName, price := range prices {
		if strings.TrimSpace(modelName) == "" {
			return nil, fmt.Errorf("audio duration price model name cannot be empty")
		}
		if price < 0 || math.IsNaN(price) || math.IsInf(price, 0) {
			return nil, fmt.Errorf("audio duration price for model %s must be a finite non-negative number", modelName)
		}
	}
	return prices, nil
}

func ValidateAudioDurationPriceJSON(jsonStr string) error {
	_, err := parseAudioDurationPrices(jsonStr)
	return err
}

func UpdateAudioDurationPriceByJSONString(jsonStr string) error {
	prices, err := parseAudioDurationPrices(jsonStr)
	if err != nil {
		return err
	}
	audioDurationPriceMap.ReplaceAll(prices)
	InvalidateExposedDataCache()
	return nil
}

func GetAudioDurationPrice(name string) (float64, bool) {
	name = FormatMatchingModelName(name)
	price, ok := audioDurationPriceMap.Get(name)
	return price, ok
}

func ContainsAudioDurationPrice(name string) bool {
	_, ok := GetAudioDurationPrice(name)
	return ok
}

func GetAudioDurationPriceCopy() map[string]float64 {
	return audioDurationPriceMap.ReadAll()
}
