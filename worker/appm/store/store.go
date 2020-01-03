// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

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

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/eapache/channels"
	"github.com/goodrain/rainbond/cmd/worker/option"
	"github.com/goodrain/rainbond/db"
	"github.com/goodrain/rainbond/db/model"
	k8sutil "github.com/goodrain/rainbond/util/k8s"
	"github.com/goodrain/rainbond/worker/appm/conversion"
	"github.com/goodrain/rainbond/worker/appm/f"
	v1 "github.com/goodrain/rainbond/worker/appm/types/v1"
	workerutil "github.com/goodrain/rainbond/worker/util"
	"github.com/jinzhu/gorm"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listcorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var rc2RecordType = map[string]string{
	"Deployment":              "mumaul",
	"Statefulset":             "mumaul",
	"HorizontalPodAutoscaler": "hpa",
}

//Storer app runtime store interface
type Storer interface {
	Start() error
	Ready() bool
	RegistAppService(*v1.AppService)
	GetAppService(serviceID string) *v1.AppService
	UpdateGetAppService(serviceID string) *v1.AppService
	GetAllAppServices() []*v1.AppService
	GetAppServiceStatus(serviceID string) string
	GetAppServicesStatus(serviceIDs []string) map[string]string
	GetTenantResource(tenantID string) *v1.TenantResource
	GetTenantRunningApp(tenantID string) []*v1.AppService
	GetNeedBillingStatus(serviceIDs []string) map[string]string
	OnDelete(obj interface{})
	GetPodLister() listcorev1.PodLister
	RegistPodUpdateListener(string, chan<- *corev1.Pod)
	UnRegistPodUpdateListener(string)
	InitOneThirdPartService(service *model.TenantServices) error
}

// EventType type of event associated with an informer
type EventType string

const (
	// CreateEvent event associated with new objects in an informer
	CreateEvent EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent EventType = "UPDATE"
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent EventType = "DELETE"
)

// Event holds the context of an event.
type Event struct {
	Type EventType
	Obj  interface{}
}

// ProbeInfo holds the context of a probe.
type ProbeInfo struct {
	Sid  string `json:"sid"`
	UUID string `json:"uuid"`
	IP   string `json:"ip"`
	Port int32  `json:"port"`
}

//appRuntimeStore app runtime store
//cache all kubernetes object and appservice
type appRuntimeStore struct {
	clientset             *kubernetes.Clientset
	ctx                   context.Context
	cancel                context.CancelFunc
	informers             *Informer
	listers               *Lister
	appServices           sync.Map
	appCount              int32
	dbmanager             db.Manager
	conf                  option.Config
	startCh               *channels.RingChannel
	stopch                chan struct{}
	podUpdateListeners    map[string]chan<- *corev1.Pod
	podUpdateListenerLock sync.Mutex
}

//NewStore new app runtime store
func NewStore(clientset *kubernetes.Clientset,
	dbmanager db.Manager,
	conf option.Config,
	startCh *channels.RingChannel,
	probeCh *channels.RingChannel) Storer {
	ctx, cancel := context.WithCancel(context.Background())
	store := &appRuntimeStore{
		clientset:          clientset,
		ctx:                ctx,
		cancel:             cancel,
		informers:          &Informer{},
		listers:            &Lister{},
		appServices:        sync.Map{},
		conf:               conf,
		dbmanager:          dbmanager,
		startCh:            startCh,
		podUpdateListeners: make(map[string]chan<- *corev1.Pod, 1),
	}
	// create informers factory, enable and assign required informers
	infFactory := informers.NewFilteredSharedInformerFactory(conf.KubeClient, time.Second, corev1.NamespaceAll,
		func(options *metav1.ListOptions) {})
	store.informers.Deployment = infFactory.Apps().V1().Deployments().Informer()
	store.listers.Deployment = infFactory.Apps().V1().Deployments().Lister()

	store.informers.StatefulSet = infFactory.Apps().V1().StatefulSets().Informer()
	store.listers.StatefulSet = infFactory.Apps().V1().StatefulSets().Lister()

	store.informers.Service = infFactory.Core().V1().Services().Informer()
	store.listers.Service = infFactory.Core().V1().Services().Lister()

	store.informers.Pod = infFactory.Core().V1().Pods().Informer()
	store.listers.Pod = infFactory.Core().V1().Pods().Lister()

	store.informers.Secret = infFactory.Core().V1().Secrets().Informer()
	store.listers.Secret = infFactory.Core().V1().Secrets().Lister()

	store.informers.ConfigMap = infFactory.Core().V1().ConfigMaps().Informer()
	store.listers.ConfigMap = infFactory.Core().V1().ConfigMaps().Lister()

	store.informers.Ingress = infFactory.Extensions().V1beta1().Ingresses().Informer()
	store.listers.Ingress = infFactory.Extensions().V1beta1().Ingresses().Lister()

	store.informers.ReplicaSet = infFactory.Apps().V1().ReplicaSets().Informer()

	store.informers.Endpoints = infFactory.Core().V1().Endpoints().Informer()
	store.listers.Endpoints = infFactory.Core().V1().Endpoints().Lister()

	store.informers.Nodes = infFactory.Core().V1().Nodes().Informer()
	store.listers.Nodes = infFactory.Core().V1().Nodes().Lister()

	store.informers.StorageClass = infFactory.Storage().V1().StorageClasses().Informer()
	store.listers.StorageClass = infFactory.Storage().V1().StorageClasses().Lister()

	store.informers.Claims = infFactory.Core().V1().PersistentVolumeClaims().Informer()
	store.listers.Claims = infFactory.Core().V1().PersistentVolumeClaims().Lister()

	store.informers.Events = infFactory.Core().V1().Events().Informer()

	store.informers.HorizontalPodAutoscaler = infFactory.Autoscaling().V2beta1().HorizontalPodAutoscalers().Informer()
	store.listers.HorizontalPodAutoscaler = infFactory.Autoscaling().V2beta1().HorizontalPodAutoscalers().Lister()

	isThirdParty := func(ep *corev1.Endpoints) bool {
		return ep.Labels["service-kind"] == model.ServiceKindThirdParty.String()
	}
	// Endpoint Event Handler
	epEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ep := obj.(*corev1.Endpoints)

			serviceID := ep.Labels["service_id"]
			version := ep.Labels["version"]
			createrID := ep.Labels["creater_id"]
			if serviceID != "" && createrID != "" {
				appservice, err := store.getAppService(serviceID, version, createrID, true)
				if err == conversion.ErrServiceNotFound {
					logrus.Debugf("ServiceID: %s; Action: AddFunc; service not found", serviceID)
				}
				if appservice != nil {
					appservice.AddEndpoints(ep)
					if isThirdParty(ep) && ep.Subsets != nil && len(ep.Subsets) > 0 {
						logrus.Debugf("received add endpoints: %+v", ep)
						probeInfos := listProbeInfos(ep, serviceID)
						probeCh.In() <- Event{
							Type: CreateEvent,
							Obj:  probeInfos,
						}
					}
					return
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			ep := obj.(*corev1.Endpoints)
			serviceID := ep.Labels["service_id"]
			version := ep.Labels["version"]
			createrID := ep.Labels["creater_id"]
			if serviceID != "" && createrID != "" {
				appservice, _ := store.getAppService(serviceID, version, createrID, false)
				if appservice != nil {
					appservice.DelEndpoints(ep)
					if appservice.IsClosed() {
						logrus.Debugf("ServiceID: %s; Action: DeleteFunc;service is closed", serviceID)
						store.DeleteAppService(appservice)
					}
					if isThirdParty(ep) {
						logrus.Debugf("received delete endpoints: %+v", ep)
						var uuids []string
						for _, item := range ep.Subsets {
							uuids = append(uuids, item.Ports[0].Name)
						}
						probeCh.In() <- Event{
							Type: DeleteEvent,
							Obj:  uuids,
						}
					}
				}
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			cep := cur.(*corev1.Endpoints)

			serviceID := cep.Labels["service_id"]
			version := cep.Labels["version"]
			createrID := cep.Labels["creater_id"]
			if serviceID != "" && createrID != "" {
				appservice, err := store.getAppService(serviceID, version, createrID, true)
				if err == conversion.ErrServiceNotFound {
					logrus.Debugf("ServiceID: %s; Action: UpdateFunc; service not found", serviceID)
				}
				if appservice != nil {
					appservice.AddEndpoints(cep)
					if isThirdParty(cep) {
						curInfos := listProbeInfos(cep, serviceID)
						probeCh.In() <- Event{
							Type: UpdateEvent,
							Obj:  curInfos,
						}
					}
				}
			}
		},
	}

	store.informers.Deployment.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.StatefulSet.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.Pod.AddEventHandlerWithResyncPeriod(store.podEventHandler(), time.Second*10)
	store.informers.Secret.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.Service.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.Ingress.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.ConfigMap.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.ReplicaSet.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.Endpoints.AddEventHandlerWithResyncPeriod(epEventHandler, time.Second*10)
	store.informers.Nodes.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	store.informers.StorageClass.AddEventHandlerWithResyncPeriod(store, time.Second*300)
	store.informers.Claims.AddEventHandlerWithResyncPeriod(store, time.Second*10)

	store.informers.Events.AddEventHandlerWithResyncPeriod(store.evtEventHandler(), time.Second*10)
	store.informers.HorizontalPodAutoscaler.AddEventHandlerWithResyncPeriod(store, time.Second*10)
	return store
}

func listProbeInfos(ep *corev1.Endpoints, sid string) []*ProbeInfo {
	var probeInfos []*ProbeInfo
	for _, subset := range ep.Subsets {
		uuid := subset.Ports[0].Name
		port := subset.Ports[0].Port
		if ep.Annotations != nil {
			if domain, ok := ep.Annotations["domain"]; ok && domain != "" {
				logrus.Debugf("thirdpart service[sid: %s] add domain endpoint[domain: %s] probe", sid, domain)
				probeInfos = []*ProbeInfo{&ProbeInfo{
					Sid:  sid,
					UUID: uuid,
					IP:   domain,
					Port: port,
				}}
				return probeInfos
			}
		}
		for _, address := range subset.NotReadyAddresses {
			info := &ProbeInfo{
				Sid:  sid,
				UUID: uuid,
				IP:   address.IP,
				Port: port,
			}
			probeInfos = append(probeInfos, info)
		}
		for _, address := range subset.Addresses {
			info := &ProbeInfo{
				Sid:  sid,
				UUID: uuid,
				IP:   address.IP,
				Port: port,
			}
			probeInfos = append(probeInfos, info)
		}
	}
	return probeInfos
}

func upgradeProbe(ch chan<- interface{}, old, cur []*ProbeInfo) {
	ob, _ := json.Marshal(old)
	cb, _ := json.Marshal(cur)
	logrus.Debugf("Old probe infos: %s", string(ob))
	logrus.Debugf("Current probe infos: %s", string(cb))
	oldMap := make(map[string]*ProbeInfo, len(old))
	for i := 0; i < len(old); i++ {
		oldMap[old[i].UUID] = old[i]
	}
	for _, c := range cur {
		if info := oldMap[c.UUID]; info != nil {
			delete(oldMap, c.UUID)
			logrus.Debugf("UUID: %s; update probe", c.UUID)
			ch <- Event{
				Type: UpdateEvent,
				Obj:  c,
			}
		} else {
			logrus.Debugf("UUID: %s; create probe", c.UUID)
			ch <- Event{
				Type: CreateEvent,
				Obj:  []*ProbeInfo{c},
			}
		}
	}
	for _, info := range oldMap {
		logrus.Debugf("UUID: %s; delete probe", info.UUID)
		ch <- Event{
			Type: DeleteEvent,
			Obj:  info,
		}
	}
}

func (a *appRuntimeStore) init() error {
	//init leader namespace
	leaderNamespace := a.conf.LeaderElectionNamespace
	if _, err := a.conf.KubeClient.CoreV1().Namespaces().Get(leaderNamespace, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			_, err = a.conf.KubeClient.CoreV1().Namespaces().Create(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: leaderNamespace,
				},
			})
		}
		if err != nil {
			return err
		}
	}
	// init third-party service
	return a.initStorageclass()
}

func (a *appRuntimeStore) Start() error {
	if err := a.init(); err != nil {
		return err
	}

	stopch := make(chan struct{})
	a.informers.Start(stopch)
	a.stopch = stopch
	go a.clean()

	for !a.Ready() {
	}
	a.initThirdPartyService()

	return nil
}

func (a *appRuntimeStore) initThirdPartyService() error {
	logrus.Debugf("begin initializing third-party services.")
	// TODO: list third party services that have open ports directly.
	svcs, err := a.dbmanager.TenantServiceDao().ListThirdPartyServices()
	if err != nil {
		logrus.Errorf("error listing third-party services: %v", err)
		return err
	}
	for _, svc := range svcs {
		if err = a.InitOneThirdPartService(svc); err != nil {
			logrus.Errorf("init thridpart service error: %v", err)
			return err
		}

		a.startCh.In() <- &v1.Event{
			Type: v1.StartEvent, // TODO: no need to distinguish between event types.
			Sid:  svc.ServiceID,
		}
	}
	return nil
}

// InitOneThirdPartService init one thridpart service
func (a *appRuntimeStore) InitOneThirdPartService(service *model.TenantServices) error {
	// ignore service without open port.
	if !a.dbmanager.TenantServicesPortDao().HasOpenPort(service.ServiceID) {
		return nil
	}

	appService, err := conversion.InitCacheAppService(a.dbmanager, service.ServiceID, "Rainbond")
	if err != nil {
		logrus.Errorf("error initializing cache app service: %v", err)
		return err
	}
	a.RegistAppService(appService)
	err = f.ApplyOne(a.clientset, appService)
	if err != nil {
		logrus.Errorf("error applying rule: %v", err)
		return err
	}
	return nil
}

//Ready if all kube informers is syncd, store is ready
func (a *appRuntimeStore) Ready() bool {
	return a.informers.Ready()
}

//checkReplicasetWhetherDelete if rs is old version,if it is old version and it always delete all pod.
// will delete it
func (a *appRuntimeStore) checkReplicasetWhetherDelete(app *v1.AppService, rs *appsv1.ReplicaSet) {
	current := app.GetCurrentReplicaSet()
	if current != nil && current.Name != rs.Name {
		//delete old version
		if v1.GetReplicaSetVersion(current) > v1.GetReplicaSetVersion(rs) {
			if rs.Status.Replicas == 0 && rs.Status.ReadyReplicas == 0 && rs.Status.AvailableReplicas == 0 {
				if err := a.conf.KubeClient.AppsV1().ReplicaSets(rs.Namespace).Delete(rs.Name, &metav1.DeleteOptions{}); err != nil && errors.IsNotFound(err) {
					logrus.Errorf("delete old version replicaset failure %s", err.Error())
				}
			}
		}
	}
}

func (a *appRuntimeStore) OnAdd(obj interface{}) {
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		serviceID := deployment.Labels["service_id"]
		version := deployment.Labels["version"]
		createrID := deployment.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetDeployment(deployment)
				return
			}
		}
	}
	if statefulset, ok := obj.(*appsv1.StatefulSet); ok {
		serviceID := statefulset.Labels["service_id"]
		version := statefulset.Labels["version"]
		createrID := statefulset.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.AppsV1().StatefulSets(statefulset.Namespace).Delete(statefulset.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetStatefulSet(statefulset)
				return
			}
		}
	}
	if replicaset, ok := obj.(*appsv1.ReplicaSet); ok {
		serviceID := replicaset.Labels["service_id"]
		version := replicaset.Labels["version"]
		createrID := replicaset.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.AppsV1().Deployments(replicaset.Namespace).Delete(replicaset.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetReplicaSets(replicaset)
				a.checkReplicasetWhetherDelete(appservice, replicaset)
				return
			}
		}
	}
	if secret, ok := obj.(*corev1.Secret); ok {
		serviceID := secret.Labels["service_id"]
		version := secret.Labels["version"]
		createrID := secret.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetSecret(secret)
				return
			}
		}
	}
	if service, ok := obj.(*corev1.Service); ok {
		serviceID := service.Labels["service_id"]
		version := service.Labels["version"]
		createrID := service.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.CoreV1().Services(service.Namespace).Delete(service.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetService(service)
				return
			}
		}
	}
	if ingress, ok := obj.(*extensions.Ingress); ok {
		serviceID := ingress.Labels["service_id"]
		version := ingress.Labels["version"]
		createrID := ingress.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.ExtensionsV1beta1().Ingresses(ingress.Namespace).Delete(ingress.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetIngress(ingress)
				return
			}
		}
	}
	if configmap, ok := obj.(*corev1.ConfigMap); ok {
		serviceID := configmap.Labels["service_id"]
		version := configmap.Labels["version"]
		createrID := configmap.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.CoreV1().ConfigMaps(configmap.Namespace).Delete(configmap.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetConfigMap(configmap)
				return
			}
		}
	}
	if hpa, ok := obj.(*v2beta1.HorizontalPodAutoscaler); ok {
		serviceID := hpa.Labels["service_id"]
		version := hpa.Labels["version"]
		createrID := hpa.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(hpa.GetNamespace()).Delete(hpa.GetName(), &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetHPA(hpa)
			}

			return
		}
	}
	if sc, ok := obj.(*storagev1.StorageClass); ok {
		vt := workerutil.TransStorageClass2RBDVolumeType(sc)
		if _, err := db.GetManager().VolumeTypeDao().CreateOrUpdateVolumeType(vt); err != nil {
			logrus.Errorf("sync storageclass error : %s, ignore it", err.Error())
		}
	}
	if claim, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		serviceID := claim.Labels["service_id"]
		version := claim.Labels["version"]
		createrID := claim.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, err := a.getAppService(serviceID, version, createrID, true)
			if err == conversion.ErrServiceNotFound {
				a.conf.KubeClient.CoreV1().PersistentVolumeClaims(claim.Namespace).Delete(claim.Name, &metav1.DeleteOptions{})
			}
			if appservice != nil {
				appservice.SetClaim(claim)
				return
			}
		}
	}
}

func (a *appRuntimeStore) listHPAEvents(hpa *v2beta1.HorizontalPodAutoscaler) error {
	namespace, name := hpa.GetNamespace(), hpa.GetName()
	eventsInterface := a.clientset.CoreV1().Events(hpa.GetNamespace())
	selector := eventsInterface.GetFieldSelector(&name, &namespace, nil, nil)
	options := metav1.ListOptions{FieldSelector: selector.String()}
	events, err := eventsInterface.List(options)
	if err != nil {
		return err
	}

	_ = events

	return nil
}

//getAppService if  creater is true, will create new app service where not found in store
func (a *appRuntimeStore) getAppService(serviceID, version, createrID string, creater bool) (*v1.AppService, error) {
	var appservice *v1.AppService
	appservice = a.GetAppService(serviceID)
	if appservice == nil && creater {
		var err error
		appservice, err = conversion.InitCacheAppService(a.dbmanager, serviceID, createrID)
		if err != nil {
			logrus.Errorf("init cache app service failure:%s", err.Error())
			return nil, err
		}
		a.RegistAppService(appservice)
	}
	return appservice, nil
}
func (a *appRuntimeStore) OnUpdate(oldObj, newObj interface{}) {
	a.OnAdd(newObj)
}
func (a *appRuntimeStore) OnDelete(obj interface{}) {
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		serviceID := deployment.Labels["service_id"]
		version := deployment.Labels["version"]
		createrID := deployment.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteDeployment(deployment)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if statefulset, ok := obj.(*appsv1.StatefulSet); ok {
		serviceID := statefulset.Labels["service_id"]
		version := statefulset.Labels["version"]
		createrID := statefulset.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteStatefulSet(statefulset)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if replicaset, ok := obj.(*appsv1.ReplicaSet); ok {
		serviceID := replicaset.Labels["service_id"]
		version := replicaset.Labels["version"]
		createrID := replicaset.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteReplicaSet(replicaset)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if secret, ok := obj.(*corev1.Secret); ok {
		serviceID := secret.Labels["service_id"]
		version := secret.Labels["version"]
		createrID := secret.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteSecrets(secret)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if service, ok := obj.(*corev1.Service); ok {
		serviceID := service.Labels["service_id"]
		version := service.Labels["version"]
		createrID := service.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteServices(service)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if ingress, ok := obj.(*extensions.Ingress); ok {
		serviceID := ingress.Labels["service_id"]
		version := ingress.Labels["version"]
		createrID := ingress.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteIngress(ingress)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if configmap, ok := obj.(*corev1.ConfigMap); ok {
		serviceID := configmap.Labels["service_id"]
		version := configmap.Labels["version"]
		createrID := configmap.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteConfigMaps(configmap)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
	if hpa, ok := obj.(*v2beta1.HorizontalPodAutoscaler); ok {
		serviceID := hpa.Labels["service_id"]
		version := hpa.Labels["version"]
		createrID := hpa.Labels["creater_id"]
		if serviceID != "" && version != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DelHPA(hpa)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}

	if sc, ok := obj.(*storagev1.StorageClass); ok {
		if err := a.dbmanager.VolumeTypeDao().DeleteModelByVolumeTypes(sc.GetName()); err != nil {
			logrus.Errorf("delete volumeType from db error: %s", err.Error())
			return
		}
	}

	if claim, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		serviceID := claim.Labels["service_id"]
		version := claim.Labels["version"]
		createrID := claim.Labels["creater_id"]
		if serviceID != "" && createrID != "" {
			appservice, _ := a.getAppService(serviceID, version, createrID, false)
			if appservice != nil {
				appservice.DeleteClaim(claim)
				if appservice.IsClosed() {
					a.DeleteAppService(appservice)
				}
				return
			}
		}
	}
}

//RegistAppService regist a app model to store.
func (a *appRuntimeStore) RegistAppService(app *v1.AppService) {
	a.appServices.Store(v1.GetCacheKeyOnlyServiceID(app.ServiceID), app)
	a.appCount++
	logrus.Debugf("current have %d app after add \n", a.appCount)
}

//DeleteAppService delete cache app service
func (a *appRuntimeStore) DeleteAppService(app *v1.AppService) {
	//a.appServices.Delete(v1.GetCacheKeyOnlyServiceID(app.ServiceID))
	//a.appCount--
	//logrus.Debugf("current have %d app after delete \n", a.appCount)
}

//DeleteAppServiceByKey delete cache app service
func (a *appRuntimeStore) DeleteAppServiceByKey(key v1.CacheKey) {
	a.appServices.Delete(key)
	a.appCount--
	logrus.Debugf("current have %d app after delete \n", a.appCount)
}

func (a *appRuntimeStore) GetAppService(serviceID string) *v1.AppService {
	key := v1.GetCacheKeyOnlyServiceID(serviceID)
	app, ok := a.appServices.Load(key)
	if ok {
		appService := app.(*v1.AppService)
		return appService
	}
	return nil
}

func (a *appRuntimeStore) UpdateGetAppService(serviceID string) *v1.AppService {
	key := v1.GetCacheKeyOnlyServiceID(serviceID)
	app, ok := a.appServices.Load(key)
	if ok {
		appService := app.(*v1.AppService)
		if statefulset := appService.GetStatefulSet(); statefulset != nil {
			stateful, err := a.listers.StatefulSet.StatefulSets(statefulset.Namespace).Get(statefulset.Name)
			if err != nil && errors.IsNotFound(err) {
				appService.DeleteStatefulSet(statefulset)
			}
			if stateful != nil {
				appService.SetStatefulSet(stateful)
			}
		}
		if deployment := appService.GetDeployment(); deployment != nil {
			deploy, err := a.listers.Deployment.Deployments(deployment.Namespace).Get(deployment.Name)
			if err != nil && errors.IsNotFound(err) {
				appService.DeleteDeployment(deployment)
			}
			if deploy != nil {
				appService.SetDeployment(deploy)
			}
		}
		if services := appService.GetServices(); services != nil {
			for _, service := range services {
				se, err := a.listers.Service.Services(service.Namespace).Get(service.Name)
				if err != nil && errors.IsNotFound(err) {
					appService.DeleteServices(service)
				}
				if se != nil {
					appService.SetService(se)
				}
			}
		}
		if ingresses := appService.GetIngress(); ingresses != nil {
			for _, ingress := range ingresses {
				in, err := a.listers.Ingress.Ingresses(ingress.Namespace).Get(ingress.Name)
				if err != nil && errors.IsNotFound(err) {
					appService.DeleteIngress(ingress)
				}
				if in != nil {
					appService.SetIngress(in)
				}
			}
		}
		if secrets := appService.GetSecrets(); secrets != nil {
			for _, secret := range secrets {
				se, err := a.listers.Secret.Secrets(secret.Namespace).Get(secret.Name)
				if err != nil && errors.IsNotFound(err) {
					appService.DeleteSecrets(secret)
				}
				if se != nil {
					appService.SetSecret(se)
				}
			}
		}
		if pods := appService.GetPods(); pods != nil {
			for _, pod := range pods {
				se, err := a.listers.Pod.Pods(pod.Namespace).Get(pod.Name)
				if err != nil && errors.IsNotFound(err) {
					appService.DeletePods(pod)
				}
				if se != nil {
					appService.SetPods(se)
				}
			}
		}

		return appService
	}
	return nil
}

func (a *appRuntimeStore) GetAllAppServices() (apps []*v1.AppService) {
	a.appServices.Range(func(k, value interface{}) bool {
		appService, _ := value.(*v1.AppService)
		if appService != nil {
			apps = append(apps, appService)
		}
		return true
	})
	return
}

func (a *appRuntimeStore) GetAppServiceStatus(serviceID string) string {
	app := a.GetAppService(serviceID)
	if app == nil {
		versions, err := a.dbmanager.VersionInfoDao().GetVersionByServiceID(serviceID)
		if (err != nil && err == gorm.ErrRecordNotFound) || len(versions) == 0 {
			return v1.UNDEPLOY
		}
		return v1.CLOSED
	}
	status := app.GetServiceStatus()
	if status == v1.UNKNOW {
		app := a.UpdateGetAppService(serviceID)
		if app == nil {
			versions, err := a.dbmanager.VersionInfoDao().GetVersionByServiceID(serviceID)
			if (err != nil && err == gorm.ErrRecordNotFound) || len(versions) == 0 {
				return v1.UNDEPLOY
			}
			return v1.CLOSED
		}
		return app.GetServiceStatus()
	}
	return status
}

func (a *appRuntimeStore) GetAppServicesStatus(serviceIDs []string) map[string]string {
	statusMap := make(map[string]string, len(serviceIDs))
	if serviceIDs == nil || len(serviceIDs) == 0 {
		a.appServices.Range(func(k, v interface{}) bool {
			appService, _ := v.(*v1.AppService)
			statusMap[appService.ServiceID] = a.GetAppServiceStatus(appService.ServiceID)
			return true
		})
		return statusMap
	}
	for _, serviceID := range serviceIDs {
		statusMap[serviceID] = a.GetAppServiceStatus(serviceID)
	}
	return statusMap
}

func (a *appRuntimeStore) GetNeedBillingStatus(serviceIDs []string) map[string]string {
	statusMap := make(map[string]string, len(serviceIDs))
	if serviceIDs == nil || len(serviceIDs) == 0 {
		a.appServices.Range(func(k, v interface{}) bool {
			appService, _ := v.(*v1.AppService)
			status := a.GetAppServiceStatus(appService.ServiceID)
			if !isClosedStatus(status) {
				statusMap[appService.ServiceID] = status
			}
			return true
		})
	} else {
		for _, serviceID := range serviceIDs {
			status := a.GetAppServiceStatus(serviceID)
			if !isClosedStatus(status) {
				statusMap[serviceID] = status
			}
		}
	}
	return statusMap
}
func isClosedStatus(curStatus string) bool {
	return curStatus == v1.BUILDEFAILURE || curStatus == v1.CLOSED || curStatus == v1.UNDEPLOY || curStatus == v1.BUILDING || curStatus == v1.UNKNOW
}

func getServiceInfoFromPod(pod *corev1.Pod) v1.AbnormalInfo {
	var ai v1.AbnormalInfo
	if len(pod.Spec.Containers) > 0 {
		var i = 0
		container := pod.Spec.Containers[0]
		for _, env := range container.Env {
			if env.Name == "SERVICE_ID" {
				ai.ServiceID = env.Value
				i++
			}
			if env.Name == "SERVICE_NAME" {
				ai.ServiceAlias = env.Value
				i++
			}
			if i == 2 {
				break
			}
		}
	}
	ai.PodName = pod.Name
	ai.TenantID = pod.Namespace
	return ai
}

func (a *appRuntimeStore) analyzePodStatus(pod *corev1.Pod) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.LastTerminationState.Terminated != nil {
			ai := getServiceInfoFromPod(pod)
			ai.ContainerName = containerStatus.Name
			ai.Reason = containerStatus.LastTerminationState.Terminated.Reason
			ai.Message = containerStatus.LastTerminationState.Terminated.Message
			ai.CreateTime = time.Now()
			a.addAbnormalInfo(&ai)
		}
	}
}

func (a *appRuntimeStore) addAbnormalInfo(ai *v1.AbnormalInfo) {
	switch ai.Reason {
	case "OOMKilled":
		a.dbmanager.NotificationEventDao().AddModel(&model.NotificationEvent{
			Kind:        "service",
			KindID:      ai.ServiceID,
			Hash:        ai.Hash(),
			Type:        "UnNormal",
			Message:     fmt.Sprintf("Container %s OOMKilled %s", ai.ContainerName, ai.Message),
			Reason:      "OOMKilled",
			Count:       ai.Count,
			ServiceName: ai.ServiceAlias,
			TenantName:  ai.TenantID,
		})
	default:
		db.GetManager().NotificationEventDao().AddModel(&model.NotificationEvent{
			Kind:        "service",
			KindID:      ai.ServiceID,
			Hash:        ai.Hash(),
			Type:        "UnNormal",
			Message:     fmt.Sprintf("Container %s restart %s", ai.ContainerName, ai.Message),
			Reason:      ai.Reason,
			Count:       ai.Count,
			ServiceName: ai.ServiceAlias,
			TenantName:  ai.TenantID,
		})
	}

}
func (a *appRuntimeStore) GetPodLister() listcorev1.PodLister {
	return a.listers.Pod
}

//GetTenantResource get tenant resource
func (a *appRuntimeStore) GetTenantResource(tenantID string) *v1.TenantResource {
	nodes, err := a.listers.Nodes.List(labels.Everything())
	if err != nil {
		logrus.Errorf("error listing nodes: %v", err)
		return nil
	}
	nodeStatus := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		nodeStatus[node.Name] = node.Spec.Unschedulable
	}
	pods, err := a.listers.Pod.Pods(tenantID).List(labels.Everything())
	if err != nil {
		logrus.Errorf("list namespace %s pod failure %s", tenantID, err.Error())
		return nil
	}

	calculateResourcesfunc := func(pod *corev1.Pod, res map[string]int64) {
		runningC := make(map[string]struct{})
		cstatus := append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...)
		for _, c := range cstatus {
			runningC[c.Name] = struct{}{}
		}
		for _, container := range pod.Spec.Containers {
			if _, ok := runningC[container.Name]; !ok {
				continue
			}
			cpulimit := container.Resources.Limits.Cpu()
			memorylimit := container.Resources.Limits.Memory()
			cpurequest := container.Resources.Requests.Cpu()
			memoryrequest := container.Resources.Requests.Memory()
			if cpulimit != nil {
				res["cpulimit"] += cpulimit.MilliValue()
			}
			if memorylimit != nil {
				if ml, ok := memorylimit.AsInt64(); ok {
					res["memlimit"] += ml
				} else {
					res["memlimit"] += memorylimit.Value()
				}
			}
			if cpurequest != nil {
				res["cpureq"] += cpurequest.MilliValue()
			}
			if memoryrequest != nil {
				if mr, ok := memoryrequest.AsInt64(); ok {
					res["memreq"] += mr
				} else {
					res["memreq"] += memoryrequest.Value()
				}
			}
		}
	}
	resource := &v1.TenantResource{}
	// schedulable resources
	sres := make(map[string]int64)
	// unschedulable resources
	ures := make(map[string]int64)
	for _, pod := range pods {
		if nodeStatus[pod.Spec.NodeName] {
			calculateResourcesfunc(pod, ures)
			continue
		}
		calculateResourcesfunc(pod, sres)
	}
	resource.CPULimit = sres["cpulimit"]
	resource.CPURequest = sres["cpureq"]
	resource.MemoryLimit = sres["memlimit"]
	resource.MemoryRequest = sres["memreq"]
	resource.UnscdCPULimit = ures["cpulimit"]
	resource.UnscdCPUReq = ures["cpureq"]
	resource.UnscdMemoryLimit = ures["memlimit"]
	resource.UnscdMemoryReq = ures["memreq"]
	return resource
}

//GetTenantRunningApp get running app by tenant
func (a *appRuntimeStore) GetTenantRunningApp(tenantID string) (list []*v1.AppService) {
	a.appServices.Range(func(k, v interface{}) bool {
		appService, _ := v.(*v1.AppService)
		if appService != nil && (appService.TenantID == tenantID || tenantID == corev1.NamespaceAll) && !appService.IsClosed() {
			list = append(list, appService)
		}
		return true
	})
	return
}

func (a *appRuntimeStore) podEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			_, serviceID, version, createrID := k8sutil.ExtractLabels(pod.GetLabels())
			if serviceID != "" && version != "" && createrID != "" {
				appservice, err := a.getAppService(serviceID, version, createrID, true)
				if err == conversion.ErrServiceNotFound {
					a.conf.KubeClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
				}
				if appservice != nil {
					appservice.SetPods(pod)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			_, serviceID, version, createrID := k8sutil.ExtractLabels(pod.GetLabels())
			if serviceID != "" && version != "" && createrID != "" {
				appservice, _ := a.getAppService(serviceID, version, createrID, false)
				if appservice != nil {
					appservice.DeletePods(pod)
					if appservice.IsClosed() {
						a.DeleteAppService(appservice)
					}
				}
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			pod := cur.(*corev1.Pod)
			_, serviceID, version, createrID := k8sutil.ExtractLabels(pod.GetLabels())
			if serviceID != "" && version != "" && createrID != "" {
				appservice, err := a.getAppService(serviceID, version, createrID, true)
				if err == conversion.ErrServiceNotFound {
					a.conf.KubeClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
				}
				if appservice != nil {
					appservice.SetPods(pod)
				}
			}
			for _, pech := range a.podUpdateListeners {
				select {
				case pech <- pod:
				default:
				}
			}
		},
	}
}

func (a *appRuntimeStore) evtEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			evt := obj.(*corev1.Event)
			recordType, ok := rc2RecordType[evt.InvolvedObject.Kind]
			if !ok {
				return
			}

			serviceID, ruleID := a.scalingRecordServiceAndRuleID(evt)
			if serviceID == "" || ruleID == "" {
				logrus.Warningf("empty service id or rule id")
				return
			}
			record := &model.TenantServiceScalingRecords{
				ServiceID:   serviceID,
				RuleID:      ruleID,
				EventName:   evt.GetName(),
				RecordType:  recordType,
				Count:       evt.Count,
				Reason:      evt.Reason,
				Description: evt.Message,
				Operator:    "system",
				LastTime:    evt.LastTimestamp.Time,
			}
			logrus.Debugf("received add record: %#v", record)

			if err := db.GetManager().TenantServiceScalingRecordsDao().UpdateOrCreate(record); err != nil {
				logrus.Warningf("update or create scaling record: %v", err)
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			oevt := old.(*corev1.Event)
			cevt := cur.(*corev1.Event)

			recordType, ok := rc2RecordType[cevt.InvolvedObject.Kind]
			if !ok {
				return
			}
			if oevt.ResourceVersion == cevt.ResourceVersion {
				return
			}

			serviceID, ruleID := a.scalingRecordServiceAndRuleID(cevt)
			if serviceID == "" || ruleID == "" {
				logrus.Warningf("empty service id or rule id")
				return
			}
			record := &model.TenantServiceScalingRecords{
				ServiceID:   serviceID,
				RuleID:      ruleID,
				EventName:   cevt.GetName(),
				RecordType:  recordType,
				Count:       cevt.Count,
				Reason:      cevt.Reason,
				LastTime:    cevt.LastTimestamp.Time,
				Description: cevt.Message,
			}
			logrus.Debugf("received update record: %#v", record)

			if err := db.GetManager().TenantServiceScalingRecordsDao().UpdateOrCreate(record); err != nil {
				logrus.Warningf("update or create scaling record: %v", err)
			}
		},
	}
}

func (a *appRuntimeStore) scalingRecordServiceAndRuleID(evt *corev1.Event) (string, string) {
	var ruleID, serviceID string
	switch evt.InvolvedObject.Kind {
	case "Deployment":
		deploy, err := a.listers.Deployment.Deployments(evt.InvolvedObject.Namespace).Get(evt.InvolvedObject.Name)
		if err != nil {
			logrus.Warningf("retrieve deployment: %v", err)
			return "", ""
		}
		serviceID = deploy.GetLabels()["service_id"]
		ruleID = deploy.GetLabels()["rule_id"]
	case "Statefulset":
		statefulset, err := a.listers.StatefulSet.StatefulSets(evt.InvolvedObject.Namespace).Get(evt.InvolvedObject.Name)
		if err != nil {
			logrus.Warningf("retrieve statefulset: %v", err)
			return "", ""
		}
		serviceID = statefulset.GetLabels()["service_id"]
		ruleID = statefulset.GetLabels()["rule_id"]
	case "HorizontalPodAutoscaler":
		hpa, err := a.listers.HorizontalPodAutoscaler.HorizontalPodAutoscalers(evt.InvolvedObject.Namespace).Get(evt.InvolvedObject.Name)
		if err != nil {
			logrus.Warningf("retrieve statefulset: %v", err)
			return "", ""
		}
		serviceID = hpa.GetLabels()["service_id"]
		ruleID = hpa.GetLabels()["rule_id"]
	default:
		logrus.Warningf("unsupported object kind: %s", evt.InvolvedObject.Kind)
	}

	return serviceID, ruleID
}

func (a *appRuntimeStore) RegistPodUpdateListener(name string, ch chan<- *corev1.Pod) {
	a.podUpdateListenerLock.Lock()
	defer a.podUpdateListenerLock.Unlock()
	a.podUpdateListeners[name] = ch
}
func (a *appRuntimeStore) UnRegistPodUpdateListener(name string) {
	a.podUpdateListenerLock.Lock()
	defer a.podUpdateListenerLock.Unlock()
	delete(a.podUpdateListeners, name)
}
