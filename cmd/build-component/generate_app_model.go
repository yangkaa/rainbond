package main

import (
	"encoding/json"
	"fmt"
	"github.com/twinj/uuid"
	"os"
	"strings"
)

const componentNum = 100

func NewUUID() string {
	uid := uuid.NewV4().String()
	return strings.Replace(uid, "-", "", -1)
}

type IngressHttpRoute struct {
	DefaultDomain        bool                   `json:"default_domain"`
	Location             string                 `json:"location"`
	Cookies              map[string]interface{} `json:"cookies"`
	Headers              map[string]interface{} `json:"headers"`
	PathRewrite          bool                   `json:"path_rewrite"`
	Rewrites             []interface{}          `json:"rewrites"`
	SSL                  bool                   `json:"ssl"`
	LoadBalancing        string                 `json:"load_balancing"`
	ConnectionTimeout    interface{}            `json:"connection_timeout"`
	RequestTimeout       interface{}            `json:"request_timeout"`
	ResponseTimeout      interface{}            `json:"response_timeout"`
	RequestBodySizeLimit interface{}            `json:"request_body_size_limit"`
	ProxyBufferNumbers   interface{}            `json:"proxy_buffer_numbers"`
	ProxyBufferSize      interface{}            `json:"proxy_buffer_size"`
	Websocket            interface{}            `json:"websocket"`
	ComponentKey         string                 `json:"component_key"`
	Port                 int                    `json:"port"`
	ProxyHeader          interface{}            `json:"proxy_header"`
}

type ServiceEnvMap struct {
	Name      string `json:"name"`
	AttrName  string `json:"attr_name"`
	AttrValue string `json:"attr_value"`
	IsChange  bool   `json:"is_change"`
}

type PortMap struct {
	Protocol       string `json:"protocol"`
	TenantID       string `json:"tenant_id"`
	PortAlias      string `json:"port_alias"`
	ContainerPort  int    `json:"container_port"`
	IsInnerService bool   `json:"is_inner_service"`
	IsOuterService bool   `json:"is_outer_service"`
	K8sServiceName string `json:"k8s_service_name"`
	Name           string `json:"name"`
}

type ComponentK8sAttribute struct {
	CreateTime     string `json:"create_time"`
	UpdateTime     string `json:"update_time"`
	TenantID       string `json:"tenant_id"`
	ComponentID    string `json:"component_id"`
	Name           string `json:"name"`
	SaveType       string `json:"save_type"`
	AttributeValue string `json:"attribute_value"`
}

type App struct {
	ServiceID                  string                  `json:"service_id"`
	TenantID                   string                  `json:"tenant_id"`
	ServiceCname               string                  `json:"service_cname"`
	ServiceKey                 string                  `json:"service_key"`
	ServiceShareUUID           string                  `json:"service_share_uuid"`
	NeedShare                  bool                    `json:"need_share"`
	Category                   string                  `json:"category"`
	Language                   string                  `json:"language"`
	ExtendMethod               string                  `json:"extend_method"`
	Version                    string                  `json:"version"`
	Memory                     int                     `json:"memory"`
	ServiceType                string                  `json:"service_type"`
	ServiceSource              string                  `json:"service_source"`
	K8sComponentName           string                  `json:"k8s_component_name"`
	DeployVersion              string                  `json:"deploy_version"`
	Image                      string                  `json:"image"`
	Arch                       string                  `json:"arch"`
	ServiceAlias               string                  `json:"service_alias"`
	ServiceName                string                  `json:"service_name"`
	ServiceRegion              string                  `json:"service_region"`
	Creater                    int                     `json:"creater"`
	Cmd                        string                  `json:"cmd"`
	Probes                     []interface{}           `json:"probes"`
	ExtendMethodMap            map[string]interface{}  `json:"extend_method_map"`
	PortMapList                []PortMap               `json:"port_map_list"`
	ServiceVolumeMapList       []interface{}           `json:"service_volume_map_list"`
	ServiceEnvMapList          []ServiceEnvMap         `json:"service_env_map_list"`
	ServiceConnectInfoMapList  []interface{}           `json:"service_connect_info_map_list"`
	ServiceRelatedPluginConfig []interface{}           `json:"service_related_plugin_config"`
	ComponentMonitors          interface{}             `json:"component_monitors"`
	ComponentGraphs            interface{}             `json:"component_graphs"`
	Labels                     map[string]interface{}  `json:"labels"`
	ComponentK8sAttributes     []ComponentK8sAttribute `json:"component_k8s_attributes"`
	DepServiceMapList          []interface{}           `json:"dep_service_map_list"`
	MntRelationList            []interface{}           `json:"mnt_relation_list"`
	ServiceImage               map[string]interface{}  `json:"service_image"`
	ShareType                  string                  `json:"share_type"`
	ShareImage                 string                  `json:"share_image"`
}

type Template struct {
	TemplateVersion   string             `json:"template_version"`
	GroupKey          string             `json:"group_key"`
	GroupName         string             `json:"group_name"`
	GroupVersion      string             `json:"group_version"`
	GroupDevStatus    string             `json:"group_dev_status"`
	GovernanceMode    string             `json:"governance_mode"`
	K8sResources      []interface{}      `json:"k8s_resources"`
	AppConfigGroups   []interface{}      `json:"app_config_groups"`
	IngressHttpRoutes []IngressHttpRoute `json:"ingress_http_routes"`
	Apps              []App              `json:"apps"`
}

func main() {
	// 初始化基础模板
	baseTemplate := Template{
		TemplateVersion: "v2",
		GroupKey:        "cc84c9ea9c8b494f86709a7a066d13a7",
		GroupName:       "100组件",
		GroupVersion:    "1.0",
		GovernanceMode:  "KUBERNETES_NATIVE_SERVICE",
		IngressHttpRoutes: []IngressHttpRoute{
			{
				DefaultDomain: true,
				Location:      "/",
				LoadBalancing: "round-robin",
				ComponentKey:  "0238ea4cd1272f7ef5ae346b05df226b",
				Port:          80,
			},
		},
	}

	// 创建组件
	for i := 1; i <= componentNum; i++ {
		serviceID := NewUUID()
		serviceKey := serviceID
		serviceCname := fmt.Sprintf("web-%d", i)
		serviceAlias := fmt.Sprintf("gr%s", strings.ToUpper(serviceID[len(serviceID)-6:]))
		//portAlias := strings.ToUpper(serviceAlias)

		app := App{
			ServiceID:        serviceID,
			TenantID:         "83d1c2f96d784656a306eebdbc0c9e0d",
			ServiceCname:     serviceCname,
			ServiceKey:       serviceKey,
			ServiceShareUUID: fmt.Sprintf("%s+%s", serviceKey, serviceKey),
			NeedShare:        true,
			Category:         "app_publish",
			ExtendMethod:     "stateless_multiple",
			Version:          "latest",
			Memory:           512,
			ServiceType:      "application",
			ServiceSource:    "docker_image",
			K8sComponentName: serviceCname,
			DeployVersion:    "20240704195537",
			Image:            "registry.cn-hangzhou.aliyuncs.com/goodrain/nginx:latest",
			Arch:             "amd64",
			ServiceAlias:     serviceAlias,
			ServiceRegion:    "dev",
			Creater:          1,
			ExtendMethodMap: map[string]interface{}{
				"step_node":     1,
				"min_memory":    64,
				"init_memory":   512,
				"max_memory":    65536,
				"step_memory":   64,
				"is_restart":    0,
				"min_node":      1,
				"container_cpu": 0,
				"max_node":      64,
			},
			PortMapList: []PortMap{
				//{
				//	Protocol:       "http",
				//	TenantID:       "83d1c2f96d784656a306eebdbc0c9e0d",
				//	PortAlias:      portAlias,
				//	ContainerPort:  80,
				//	IsInnerService: false,
				//	IsOuterService: true,
				//	K8sServiceName: serviceAlias,
				//},
			},
			ServiceEnvMapList: []ServiceEnvMap{
				{"NGINX_VERSION", "NGINX_VERSION", "1.17.10", true},
				{"NJS_VERSION", "NJS_VERSION", "0.3.9", true},
				{"PKG_RELEASE", "PKG_RELEASE", "1~buster", true},
			},
			ComponentK8sAttributes: []ComponentK8sAttribute{
				{
					CreateTime:     "2024-07-04 19:55:37",
					UpdateTime:     "2024-07-04 19:55:37",
					TenantID:       "83d1c2f96d784656a306eebdbc0c9e0d",
					ComponentID:    serviceID,
					Name:           "affinity",
					SaveType:       "yaml",
					AttributeValue: "nodeAffinity:\n  requiredDuringSchedulingIgnoredDuringExecution:\n    nodeSelectorTerms:\n    - matchExpressions:\n      - key: kubernetes.io/arch\n        operator: In\n        values:\n        - amd64\n",
				},
			},
			ServiceImage:               map[string]interface{}{},
			ShareType:                  "image",
			ShareImage:                 "registry.cn-hangzhou.aliyuncs.com/goodrain/nginx:latest",
			ServiceConnectInfoMapList:  []interface{}{},
			DepServiceMapList:          []interface{}{},
			MntRelationList:            []interface{}{},
			ServiceRelatedPluginConfig: []interface{}{},
		}

		baseTemplate.Apps = append(baseTemplate.Apps, app)
	}

	// 序列化为JSON
	output, err := json.MarshalIndent(baseTemplate, "", "  ")
	if err != nil {
		panic(err)
	}

	// 输出到文件
	err = os.WriteFile("components.json", output, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("components.json 文件已生成")
}
