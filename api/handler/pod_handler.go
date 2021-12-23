package handler

import (
	"github.com/goodrain/rainbond/api/model"
	"github.com/goodrain/rainbond/worker/client"
	"github.com/goodrain/rainbond/worker/server/pb"
	"k8s.io/client-go/kubernetes"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

// PodHandler defines handler methods about k8s pods.
type PodHandler interface {
	PodDetail(namespace, podName string) (*pb.PodDetail, error)
	InstancesMonitor(nodeName, query string) ([]*model.PodResourceStatus, error)
}

// NewPodHandler creates a new PodHandler.
func NewPodHandler(statusCli *client.AppRuntimeSyncClient, clientset *kubernetes.Clientset, metricClient *metrics.Clientset) PodHandler {
	return &PodAction{
		statusCli:    statusCli,
		clientset:    clientset,
		metricClient: metricClient,
	}
}
