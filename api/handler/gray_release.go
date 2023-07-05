package handler

import (
	"context"
	"encoding/json"
	apimodel "github.com/goodrain/rainbond/api/model"
	"github.com/goodrain/rainbond/db"
	"github.com/goodrain/rainbond/db/model"
	"github.com/goodrain/rainbond/worker/appm/conversion"
	"github.com/jinzhu/gorm"
	"github.com/openkruise/kruise-api/rollouts/v1alpha1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (a *ApplicationAction) GetAppGrayscaleRelease(ctx context.Context, appID, componentID, namespace string) ([]apimodel.GrayReleaseModeRet, error) {
	var rollouts *v1alpha1.RolloutList
	var err error
	if componentID != "" {
		rollouts, err = a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{LabelSelector: "component_id=" + componentID})
	} else {
		rollouts, err = a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app_id=" + appID})
	}
	istioNamespace := "istio-system"
	istioServices, err := a.kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: "service-name=istio-istio"})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	if istioServices != nil && len(istioServices.Items) > 0 {
		istioNamespace = istioServices.Items[0].GetNamespace()
	}
	if err != nil {
		return nil, err
	}
	componentVersion, err := a.GrayscaleReleaseComponentVersion(ctx, rollouts, appID, namespace)
	if err != nil {
		return nil, err
	}
	var grayData []apimodel.GrayReleaseModeRet
	for _, rollout := range rollouts.Items {
		if rollout.Status.CanaryStatus != nil {
			step := len(rollout.Spec.Strategy.Canary.Steps)
			var currentStepIndex int
			if rollout.Status.Conditions != nil {
				currentStepIndex = int(rollout.Status.CanaryStatus.CurrentStepIndex)
			}
			componentID = rollout.Labels["component_id"]
			version := componentVersion[componentID]
			stepState := ""
			switch rollout.Status.CanaryStatus.CurrentStepState {
			case v1alpha1.CanaryStepStateUpgrade:
				stepState = "步骤升级"
			case v1alpha1.CanaryStepStateReady:
				stepState = "准备就绪"
			case v1alpha1.CanaryStepStateCompleted:
				stepState = "完成灰度"
			default:
				stepState = "步骤暂停"
			}
			grayData = append(grayData, apimodel.GrayReleaseModeRet{
				ComponentID:         componentID,
				Hostname:            rollout.Labels["hostname"],
				IstioNamespace:      istioNamespace,
				CanaryReadyReplicas: rollout.Status.CanaryStatus.CanaryReadyReplicas,
				CanaryReplicas:      rollout.Status.CanaryStatus.CanaryReplicas,
				CurrentStepIndex:    currentStepIndex,
				CurrentStepState:    stepState,
				Message:             rollout.Status.Message,
				Step:                step,
				NewVersion:          version["new_version"],
				OldVersion:          version["old_version"],
			})
		}
	}
	return grayData, nil
}

func (a *ApplicationAction) AddAppGrayscaleRelease(gr *apimodel.GrayReleaseModeReq) (string, error) {
	jsonFlowEntryRule, err := json.Marshal(gr.FlowEntryRule)
	if err != nil {
		return "", err
	}
	jsonGrayStrategy, err := json.Marshal(gr.GrayStrategy)
	if err != nil {
		return "", err
	}
	wasmResource := schema.GroupVersionResource{Group: "extensions.istio.io", Version: "v1alpha1", Resource: "wasmplugins"}
	app, err := db.GetManager().ApplicationDao().GetAppByID(gr.AppID)
	if err != nil {
		return "", err
	}
	//创建 WasmPlugin
	wasmObj := a.generateWasmObj(gr.AppID, app.K8sApp, gr.Namespace, string(jsonFlowEntryRule), gr.TraceType)
	finalWasm, err := a.dynamicClient.Resource(wasmResource).Namespace(gr.Namespace).Create(context.Background(), wasmObj, metav1.CreateOptions{})
	if err != nil && !k8sErrors.IsAlreadyExists(err) {
		return "", err
	}
	wasmYaml, err := a.handleDBK8sResource(app, finalWasm, "create", "WasmPlugin")
	if err != nil {
		return "", err
	}
	gray := model.AppGrayRelease{
		AppID:            gr.AppID,
		EntryComponentID: gr.EntryComponentID,
		EntryHTTPRoute:   gr.EntryHttpRoute,
		FlowEntryRule:    string(jsonFlowEntryRule),
		GrayStrategyType: gr.GrayStrategyType,
		GrayStrategy:     string(jsonGrayStrategy),
		Status:           gr.Status,
		TraceType:        gr.TraceType,
	}
	err = db.GetManager().AppGrayReleaseDao().AddModel(&gray)
	if err != nil {
		return "", err
	}
	return wasmYaml, nil
}

func (a *ApplicationAction) UpdateAppGrayscaleRelease(ctx context.Context, gray *apimodel.GrayReleaseModeReq) (string, error) {
	jsonFlowEntryRule, err := json.Marshal(gray.FlowEntryRule)
	if err != nil {
		return "", err
	}
	jsonGrayStrategy, err := json.Marshal(gray.GrayStrategy)
	if err != nil {
		return "", err
	}
	grayDB, err := db.GetManager().AppGrayReleaseDao().GetGrayRelease(gray.AppID)
	if err != nil {
		return "", err
	}
	grayDB.EntryComponentID = gray.EntryComponentID
	grayDB.FlowEntryRule = string(jsonFlowEntryRule)
	grayDB.GrayStrategyType = gray.GrayStrategyType
	grayDB.GrayStrategy = string(jsonGrayStrategy)
	grayDB.EntryHTTPRoute = gray.EntryHttpRoute
	grayDB.TraceType = gray.TraceType
	wasmResource := schema.GroupVersionResource{Group: "extensions.istio.io", Version: "v1alpha1", Resource: "wasmplugins"}
	app, err := db.GetManager().ApplicationDao().GetAppByID(gray.AppID)
	if err != nil {
		return "", err
	}
	updateWasm, err := a.dynamicClient.Resource(wasmResource).Namespace(gray.Namespace).Get(context.Background(), app.K8sApp, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsAlreadyExists(err) {
		return "", err
	}
	updateWasm.Object["spec"] = map[string]interface{}{
		"selector": map[string]interface{}{
			"matchLabels": map[string]interface {
			}{
				"app_id": gray.AppID,
			},
		},
		"url": a.conf.WasmURL,
		"vmConfig": map[string]interface{}{
			"env": []map[string]interface{}{
				{
					"name":  "GrayHeader",
					"value": grayDB.FlowEntryRule,
				}, {
					"name":  "TraceType",
					"value": grayDB.TraceType,
				},
			},
		},
	}
	finalWasm, err := a.dynamicClient.Resource(wasmResource).Namespace(gray.Namespace).Update(context.Background(), updateWasm, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}
	wasmYaml, err := a.handleDBK8sResource(app, finalWasm, "update", "WasmPlugin")
	if err != nil {
		return "", err
	}
	rollouts, err := a.kruiseClient.RolloutsV1alpha1().Rollouts(gray.Namespace).List(ctx, metav1.ListOptions{LabelSelector: "app_id=" + gray.AppID})
	if err != nil {
		return "", err
	}
	for _, rollout := range rollouts.Items {
		trafficRoutings := rollout.Spec.Strategy.Canary.TrafficRoutings
		var currentStepIndex int
		if rollout.Status.Conditions != nil {
			currentStepIndex = int(rollout.Status.CanaryStatus.CurrentStepIndex)
		}
		steps := rollout.Spec.Strategy.Canary.Steps[:currentStepIndex]
		rollout.Spec, err = conversion.HandleRolloutSpec(&grayDB, "", "", rollout.Name)
		if err != nil {
			return "", err
		}
		for i, step := range steps {
			rollout.Spec.Strategy.Canary.Steps[i] = step
		}
		rollout.Spec.Strategy.Canary.TrafficRoutings = trafficRoutings
		_, err := a.kruiseClient.RolloutsV1alpha1().Rollouts(gray.Namespace).Update(ctx, &rollout, metav1.UpdateOptions{})
		if err != nil {
			return "", err
		}
	}
	err = db.GetManager().AppGrayReleaseDao().UpdateModel(&grayDB)
	if err != nil {
		return "", err
	}
	return wasmYaml, nil
}

func (a *ApplicationAction) CloseAppGrayscaleRelease(ctx context.Context, appID, namespace string) error {
	gray, err := db.GetManager().AppGrayReleaseDao().GetGrayRelease(appID)
	if err != nil {
		return err
	}
	wasmResource := schema.GroupVersionResource{Group: "extensions.istio.io", Version: "v1alpha1", Resource: "wasmplugins"}
	app, err := db.GetManager().ApplicationDao().GetAppByID(appID)
	if err != nil {
		return err
	}
	err = a.dynamicClient.Resource(wasmResource).Namespace(namespace).Delete(context.Background(), app.K8sApp, metav1.DeleteOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	_, err = a.handleDBK8sResource(app, nil, "delete", "WasmPlugin")
	if err != nil {
		return err
	}
	err = a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "app_id=" + appID})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return err
	}
	gray.Status = false
	return db.GetManager().AppGrayReleaseDao().UpdateModel(&gray)
}

func (a *ApplicationAction) OpenAppGrayscaleRelease(appID, namespace string) (string, error) {
	grayDB, err := db.GetManager().AppGrayReleaseDao().GetGrayRelease(appID)
	if err != nil {
		return "", err
	}
	wasmResource := schema.GroupVersionResource{Group: "extensions.istio.io", Version: "v1alpha1", Resource: "wasmplugins"}
	app, err := db.GetManager().ApplicationDao().GetAppByID(appID)
	if err != nil {
		return "", err
	}
	//创建 WasmPlugin
	wasmObj := a.generateWasmObj(appID, app.K8sApp, namespace, grayDB.FlowEntryRule, grayDB.TraceType)
	finalWasm, err := a.dynamicClient.Resource(wasmResource).Namespace(namespace).Create(context.Background(), wasmObj, metav1.CreateOptions{})
	if err != nil && !k8sErrors.IsAlreadyExists(err) {
		return "", err
	}
	grayDB.Status = true
	wasmYaml, err := a.handleDBK8sResource(app, finalWasm, "create", "WasmPlugin")
	if err != nil {
		return "", err
	}
	err = db.GetManager().AppGrayReleaseDao().UpdateModel(&grayDB)
	if err != nil {
		return "", err
	}
	return wasmYaml, nil
}

func (a *ApplicationAction) NextBatchAppGrayscaleRelease(ctx context.Context, appID, namespace string) error {
	rollouts, err := a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app_id=" + appID})
	if err != nil {
		return err
	}
	for _, rollout := range rollouts.Items {
		rollout.Status.CanaryStatus.CurrentStepState = v1alpha1.CanaryStepStateReady
		_, err := a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).UpdateStatus(ctx, &rollout, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *ApplicationAction) RollBackAppGrayscaleRelease(ctx context.Context, appID, namespace string) (map[string]map[string]string, error) {
	rollouts, err := a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app_id=" + appID})
	if err != nil {
		return nil, err
	}
	componentVersion, err := a.GrayscaleReleaseComponentVersion(ctx, rollouts, appID, namespace)
	if err != nil {
		return nil, err
	}
	a.RollBackApp(componentVersion)
	err = a.kruiseClient.RolloutsV1alpha1().Rollouts(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "app_id=" + appID})
	if err != nil {
		return nil, err
	}
	return componentVersion, nil
}

func (a *ApplicationAction) RollBackApp(componentVersion map[string]map[string]string) {
	for componentID, version := range componentVersion {
		rollback := apimodel.RollbackInfoRequestStruct{
			RollBackVersion: version["old_version"],
			ServiceID:       componentID,
		}
		GetOperationHandler().RollBack(rollback)
	}

}

func (a *ApplicationAction) GrayscaleReleaseComponentVersion(ctx context.Context, rollouts *v1alpha1.RolloutList, appID, namespace string) (map[string]map[string]string, error) {
	pods, err := a.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app_id=" + appID})
	if err != nil {
		return nil, err
	}
	podList := make(map[string][]v1.Pod)
	for _, pod := range pods.Items {
		p := pod
		componentID := pod.Labels["service_id"]
		componentPods, ok := podList[componentID]
		if ok {
			podList[componentID] = append(componentPods, p)
		} else {
			podList[componentID] = []v1.Pod{p}
		}
	}
	componentVersion := make(map[string]map[string]string)
	for _, rollout := range rollouts.Items {
		componentID := rollout.Labels["component_id"]
		pods, ok := podList[componentID]
		if ok && len(pods) > 0 {
			version := pods[0].Labels["version"]
			var oldVersion, newVersion string
			for _, pod := range pods {
				if pod.Labels["version"] != oldVersion {
					if version < pod.Labels["version"] {
						oldVersion = version
						newVersion = pod.Labels["version"]
					} else {
						oldVersion = pod.Labels["version"]
						newVersion = version
					}
				}
			}
			versionMap := make(map[string]string)
			versionMap["old_version"] = oldVersion
			versionMap["new_version"] = newVersion
			componentVersion[componentID] = versionMap
		}
	}

	return componentVersion, nil
}
