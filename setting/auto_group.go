package setting

import (
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var autoGroups = []string{
	"default",
}
var autoGroupsMutex sync.RWMutex

var DefaultUseAutoGroup = false

func ContainsAutoGroup(group string) bool {
	autoGroupsMutex.RLock()
	defer autoGroupsMutex.RUnlock()
	for _, autoGroup := range autoGroups {
		if autoGroup == group {
			return true
		}
	}
	return false
}

func UpdateAutoGroupsByJsonString(jsonString string) error {
	autoGroupsMutex.Lock()
	defer autoGroupsMutex.Unlock()
	autoGroups = make([]string, 0)
	return common.Unmarshal([]byte(jsonString), &autoGroups)
}

func AutoGroups2JsonString() string {
	autoGroupsMutex.RLock()
	defer autoGroupsMutex.RUnlock()
	jsonBytes, err := common.Marshal(autoGroups)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func GetAutoGroups() []string {
	autoGroupsMutex.RLock()
	defer autoGroupsMutex.RUnlock()
	copyGroups := make([]string, len(autoGroups))
	copy(copyGroups, autoGroups)
	return copyGroups
}
