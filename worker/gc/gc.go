// Copyright (C) 2nilfmt.Errorf("a")4-2nilfmt.Errorf("a")8 Goodrain Co., Ltd.
// RAINBOND, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package gc

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond/db"
	utils "github.com/goodrain/rainbond/util"
	"os"
	"path"
	"time"

	eventutil "github.com/goodrain/rainbond/eventlog/util"
	"github.com/goodrain/rainbond/worker/discover/model"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// GarbageCollector -
type GarbageCollector struct {
	clientset kubernetes.Interface
}

// NewGarbageCollector -
func NewGarbageCollector(clientset kubernetes.Interface) *GarbageCollector {
	gcr := &GarbageCollector{
		clientset: clientset,
	}
	return gcr
}

// DelLogFile deletes persistent data related to the service based on serviceID.
func (g *GarbageCollector) DelLogFile(serviceGCReq model.ServiceGCTaskBody) {
	logrus.Infof("service id: %s; delete log file.", serviceGCReq.ServiceID)
	// log generated during service running
	logPath := "/grdata/logs"
	dockerLogPath := eventutil.DockerLogFilePath(logPath, serviceGCReq.ServiceID)
	if err := os.RemoveAll(dockerLogPath); err != nil {
		logrus.Warningf("remove docker log files: %v", err)
	}
	// log generated by the service event
	eventLogPath := eventutil.EventLogFilePath(logPath)
	for _, eventID := range serviceGCReq.EventIDs {
		eventLogFileName := eventutil.EventLogFileName(eventLogPath, eventID)
		logrus.Debugf("remove event log file: %s", eventLogFileName)
		if err := os.RemoveAll(eventLogFileName); err != nil {
			logrus.Warningf("file: %s; remove event log file: %v", eventLogFileName, err)
		}
	}
}

// DelVolumeData -
func (g *GarbageCollector) DelVolumeData(serviceGCReq model.ServiceGCTaskBody) {
	f := func(prefix string) {
		dir := path.Join(prefix, fmt.Sprintf("tenant/%s/service/%s", serviceGCReq.TenantID, serviceGCReq.ServiceID))
		logrus.Infof("volume data. delete %s", dir)
		if err := os.RemoveAll(dir); err != nil {
			logrus.Warningf("dir: %s; remove volume data: %v", dir, err)
		}
	}
	f("/grdata")
}

// DelPvPvcByServiceID -
func (g *GarbageCollector) DelPvPvcByServiceID(serviceGCReq model.ServiceGCTaskBody) {
	logrus.Infof("service_id: %s", serviceGCReq.ServiceID)
	deleteOpts := metav1.DeleteOptions{}
	listOpts := g.listOptionsServiceID(serviceGCReq.ServiceID)
	namespace := serviceGCReq.TenantID
	tenant, err := db.GetManager().TenantDao().GetTenantByUUID(serviceGCReq.TenantID)
	if err != nil {
		logrus.Warningf("tenant id: %s; get tenant before delete a collection for PV: %v", serviceGCReq.TenantID, err)
	}
	if tenant != nil {
		namespace = tenant.Namespace
	}
	if err := g.clientset.CoreV1().PersistentVolumes().DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("service id: %s; delete a collection for PV: %v", serviceGCReq.ServiceID, err)
	}

	if err := g.clientset.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("service id: %s; delete a collection for PVC: %v", serviceGCReq.ServiceID, err)
	}
}

// DelKubernetesObjects deletes all kubernetes objects.
func (g *GarbageCollector) DelKubernetesObjects(serviceGCReq model.ServiceGCTaskBody) {
	deleteOpts := metav1.DeleteOptions{}
	listOpts := g.listOptionsServiceID(serviceGCReq.ServiceID)
	namespace := serviceGCReq.TenantID
	tenant, err := db.GetManager().TenantDao().GetTenantByUUID(serviceGCReq.TenantID)
	if err != nil {
		logrus.Warningf("[DelKubernetesObjects] get tenant(%s): %v", serviceGCReq.TenantID, err)
	}
	if tenant != nil {
		namespace = tenant.Namespace
	}
	if err := g.clientset.AppsV1().Deployments(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete deployments(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.AppsV1().StatefulSets(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete statefulsets(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.BatchV1().Jobs(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete job(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.BatchV1beta1().CronJobs(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete cronjob(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.BatchV1().CronJobs(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete cronjob(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.ExtensionsV1beta1().Ingresses(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete extensions ingresses(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.NetworkingV1().Ingresses(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete networking ingresses(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.CoreV1().Secrets(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete secrets(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.CoreV1().ConfigMaps(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete configmaps(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete hpas(%s): %v", serviceGCReq.ServiceID, err)
	}
	if err := g.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete hpas(%s): %v", serviceGCReq.ServiceID, err)
	}
	// kubernetes does not support api for deleting collection of service
	// read: https://github.com/kubernetes/kubernetes/issues/68468#issuecomment-419981870
	serviceList, err := g.clientset.CoreV1().Services(namespace).List(context.Background(), listOpts)
	if err != nil {
		logrus.Warningf("[DelKubernetesObjects] list services(%s): %v", serviceGCReq.ServiceID, err)
	} else {
		for _, svc := range serviceList.Items {
			if err := g.clientset.CoreV1().Services(namespace).Delete(context.Background(), svc.Name, deleteOpts); err != nil {
				logrus.Warningf("[DelKubernetesObjects] delete service(%s): %v", svc.GetName(), err)
			}
		}
	}
	// delete endpoints after deleting services
	if err := g.clientset.CoreV1().Endpoints(namespace).DeleteCollection(context.Background(), deleteOpts, listOpts); err != nil {
		logrus.Warningf("[DelKubernetesObjects] delete endpoints(%s): %v", serviceGCReq.ServiceID, err)
	}
}

// listOptionsServiceID -
func (g *GarbageCollector) listOptionsServiceID(serviceID string) metav1.ListOptions {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
		"creator":    "Rainbond",
		"service_id": serviceID,
	}}
	return metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
}

// DelComponentPkg deletes component package
func (g *GarbageCollector) DelComponentPkg(serviceGCReq model.ServiceGCTaskBody) {
	logrus.Infof("service id: %s; delete component package.", serviceGCReq.ServiceID)
	// log generated during service running
	pkgPath := fmt.Sprintf("/grdata/package_build/components/%s", serviceGCReq.ServiceID)
	if err := os.RemoveAll(pkgPath); err != nil {
		logrus.Warningf("remove component package: %v", err)
	}
}

// DelShellPod deletes shell pod
func (g *GarbageCollector) DelShellPod() {
	podList, err := g.clientset.CoreV1().Pods(utils.GetenvDefault("RBD_NAMESPACE", constants.Namespace)).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=shell-tool",
	})
	if err != nil {
		logrus.Error("get shell pods error:", err)
	}
	for _, pod := range podList.Items {
		duration := time.Since(pod.CreationTimestamp.Time)
		if duration.Hours() > 6 {
			err = g.clientset.CoreV1().Pods(utils.GetenvDefault("RBD_NAMESPACE", constants.Namespace)).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			if err != nil {
				logrus.Error("delete shell pod error:", err)
			}
		}
	}
}
