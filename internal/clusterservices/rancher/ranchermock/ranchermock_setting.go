package ranchermock

import (
	"errors"

	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockSetting struct {
	managementv3.SettingOperations
	settings map[string]*managementv3.Setting
}

func (s *MockSetting) ByID(id string) (*managementv3.Setting, error) {
	setting, hasSetting := s.settings[id]
	if !hasSetting {
		return nil, errors.New("no such setting")
	}

	return setting, nil
}

var _ managementv3.SettingOperations = &MockSetting{}
