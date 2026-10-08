package dto

import (
	"errors"
	"math"
)

// CacheCreation separates TTL evidence from the unclassified remainder.
// Total already includes FiveMinute and OneHour; never add it to them again.
type CacheCreation struct {
	Total        int
	FiveMinute   int
	OneHour      int
	Unclassified int
}

func ResolveCacheCreation(total, fiveMinute, oneHour int, totalReported bool) (CacheCreation, error) {
	if total < 0 || fiveMinute < 0 || oneHour < 0 {
		return CacheCreation{}, errors.New("negative cache creation usage")
	}
	if fiveMinute > math.MaxInt-oneHour {
		return CacheCreation{}, errors.New("cache creation usage overflow")
	}
	split := fiveMinute + oneHour
	if total == 0 && !totalReported {
		total = split
	}
	if split > total {
		return CacheCreation{}, errors.New("cache creation breakdown exceeds total")
	}
	return CacheCreation{Total: total, FiveMinute: fiveMinute, OneHour: oneHour, Unclassified: total - split}, nil
}

func (u *Usage) CacheCreationBreakdown() (CacheCreation, error) {
	if u == nil {
		return CacheCreation{}, nil
	}
	return ResolveCacheCreation(u.PromptTokensDetails.CachedCreationTokens,
		u.ClaudeCacheCreation5mTokens, u.ClaudeCacheCreation1hTokens, u.CacheCreationTotalReported)
}
