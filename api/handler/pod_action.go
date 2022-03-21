package handler

import (
	"context"
	"github.com/sirupsen/logrus"

	"github.com/goodrain/rainbond/api/model"
	"github.com/goodrain/rainbond/worker/server/pb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"strings"

	"github.com/goodrain/rainbond/worker/client"
	"github.com/goodrain/rainbond/worker/server"
)

// PodAction is an implementation of PodHandler
type PodAction struct {
	statusCli    *client.AppRuntimeSyncClient
	clientset    *kubernetes.Clientset
	metricClient *metrics.Clientset
}

// PodDetail -
func (p *PodAction) PodDetail(namespace, podName string) (*pb.PodDetail, error) {
	pd, err := p.statusCli.GetPodDetail(namespace, podName)
	if err != nil {
		if strings.Contains(err.Error(), server.ErrPodNotFound.Error()) {
			return nil, server.ErrPodNotFound
		}
		return nil, err
	}
	return pd, nil
}

// InstancesMonitor -
func (p *PodAction) InstancesMonitor(nodeName, query string) ([]*model.PodResourceStatus, error) {
	var componentPods []*model.PodResourceStatus
	listOptions := metav1.ListOptions{LabelSelector: "creator=Rainbond"}
	if nodeName != "" {
		listOptions.FieldSelector = "spec.nodeName=" + nodeName
	}
	pods, err := p.clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), listOptions)
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		if _, ok := pod.Labels["service_id"]; !ok {
			continue
		}
		if query == "unhealthy" && pod.Status.Phase == corev1.PodRunning {
			continue
		}
		cpuUsage, memoryUsage := p.getPodQuantity(pod.Namespace, pod.Name)
		componentPods = append(componentPods, &model.PodResourceStatus{
			Node:          pod.Spec.NodeName,
			TenantID:      pod.Labels["tenant_id"],
			AppID:         pod.Labels["app_id"],
			ComponentID:   pod.Labels["service_id"],
			CPUUsage:      cpuUsage,
			MemoryUsage:   memoryUsage,
			Status:        string(pod.Status.Phase),
			Kind:          p.getPodOwnerKind(pod),
			PodNameSuffix: strings.Split(pod.Name, "-")[len(strings.Split(pod.Name, "-"))-1],
		})
	}
	return componentPods, nil
}

func (p *PodAction) getPodQuantity(namespace, podName string) (cpu, memory int64) {
	podMetrics, err := p.metricClient.MetricsV1beta1().PodMetricses(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("get pod [%s] resource quantity failed: [%v]", podName, err)
		return 0, 0
	}
	for _, container := range podMetrics.Containers {
		cpu += container.Usage.Cpu().MilliValue()
		memory += container.Usage.Memory().Value()
	}
	return cpu, memory
}

func (p *PodAction) getPodOwnerKind(pod corev1.Pod) string {
	for _, owner := range pod.OwnerReferences {
		switch owner.Kind {
		// TODO: Constant definition using k8s
		case "StatefulSet":
			return "StatefulSet"
		case "ReplicaSet":
			return "Deployment"
		}
	}
	return ""
}
