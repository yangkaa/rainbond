package kubernetesnative

import (
	"github.com/goodrain/rainbond/api/app_governance_mode/adaptor"
)

type kubernetesNativeMode struct {
}

// New Kubernetes Native Mode Handler
func New() adaptor.AppGoveranceModeHandler {
	return &kubernetesNativeMode{}
}

// IsInstalledControlPlane -
func (k *kubernetesNativeMode) IsInstalledControlPlane() bool {
	return true
}

// GetInjectLabels -
func (k *kubernetesNativeMode) GetInjectLabels() map[string]string {
	return nil
}
