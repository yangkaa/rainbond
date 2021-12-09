package buildin

import (
	"github.com/goodrain/rainbond/api/app_governance_mode/adaptor"
)

type buildInServiceMeshMode struct{}

// New Build In ServiceMeshMode Handler
func New() adaptor.AppGoveranceModeHandler {
	return &buildInServiceMeshMode{}
}

// IsInstalledControlPlane -
func (b *buildInServiceMeshMode) IsInstalledControlPlane() bool {
	return true
}

// GetInjectLabels -
func (b *buildInServiceMeshMode) GetInjectLabels() map[string]string {
	return nil
}
