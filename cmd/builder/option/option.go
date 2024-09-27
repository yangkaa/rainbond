// Copyright (C) 2014-2018 Goodrain Co., Ltd.
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

package option

import (
	"fmt"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond/builder/sources"
	"github.com/goodrain/rainbond/mq/client"
	utils "github.com/goodrain/rainbond/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"runtime"
)

// Config config server
type Config struct {
	ClusterName          string
	MysqlConnectionInfo  string
	BuildKitImage        string
	BuildKitArgs         string
	BuildKitCache        bool
	DBType               string
	PrometheusMetricPath string
	EventLogServers      []string
	KubeConfig           string
	MaxTasks             int
	APIPort              int
	MQAPI                string
	DockerEndpoint       string
	HostIP               string
	CleanUp              bool
	Topic                string
	LogPath              string
	RbdNamespace         string
	RbdRepoName          string
	GRDataPVCName        string
	CachePVCName         string
	CacheMode            string
	CachePath            string
	ContainerRuntime     string
	RuntimeEndpoint      string
	KeepCount            int
	CleanInterval        int
	BRVersion            string

	ElasticSearchURL      string
	ElasticSearchUsername string
	ElasticSearchPassword string
	ElasticEnable         bool
}

// Builder  builder server
type Builder struct {
	Config
	LogLevel string
	RunMode  string //default,sync
}

// NewBuilder new server
func NewBuilder() *Builder {
	return &Builder{}
}

// AddFlags config
func (a *Builder) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.LogLevel, "log-level", "info", "the builder log level")
	fs.StringVar(&a.PrometheusMetricPath, "metric", "/metrics", "prometheus metrics path")
	fs.StringVar(&a.BuildKitImage, "buildkit-image", "registry.cn-hangzhou.aliyuncs.com/goodrain/buildkit:v0.12.0", "buildkit image version")
	fs.StringVar(&a.DBType, "db-type", "mysql", "db type mysql or etcd")
	fs.StringVar(&a.MysqlConnectionInfo, "mysql", "root:admin@tcp(127.0.0.1:3306)/region", "mysql db connection info")
	fs.StringVar(&a.KubeConfig, "kube-config", "", "kubernetes api server config file")
	fs.IntVar(&a.MaxTasks, "max-tasks", 50, "Maximum number of simultaneous build tasks")
	fs.IntVar(&a.APIPort, "api-port", 3228, "the port for api server")
	fs.StringVar(&a.RunMode, "run", "sync", "sync data when worker start")
	fs.StringVar(&a.DockerEndpoint, "dockerd", "127.0.0.1:2376", "dockerd endpoint")
	fs.StringVar(&a.HostIP, "hostIP", "", "Current node Intranet IP")
	fs.BoolVar(&a.CleanUp, "clean-up", true, "Turn on build version cleanup")
	fs.StringVar(&a.Topic, "topic", "builder", "Topic in mq,you coule choose `builder` or `windows_builder`")
	fs.StringVar(&a.LogPath, "log-path", "/grdata/logs", "Where Docker log files and event log files are stored.")
	fs.StringVar(&a.RbdNamespace, "rbd-namespace", utils.GetenvDefault("RBD_NAMESPACE", constants.Namespace), "rbd component namespace")
	fs.StringVar(&a.RbdRepoName, "rbd-repo", "rbd-repo", "rbd component repo's name")
	fs.StringVar(&a.GRDataPVCName, "pvc-grdata-name", "grdata", "pvc name of grdata")
	fs.StringVar(&a.CachePVCName, "pvc-cache-name", "cache", "pvc name of cache")
	fs.StringVar(&a.CacheMode, "cache-mode", "sharefile", "volume cache mount type, can be hostpath and sharefile, default is sharefile, which mount using pvc")
	fs.StringVar(&a.CachePath, "cache-path", "/cache", "volume cache mount path, when cache-mode using hostpath, default path is /cache")
	fs.StringVar(&a.ContainerRuntime, "container-runtime", sources.ContainerRuntimeContainerd, "container runtime, support docker and containerd")
	fs.StringVar(&a.RuntimeEndpoint, "runtime-endpoint", sources.RuntimeEndpointContainerd, "container runtime endpoint")
	fs.StringVar(&a.BuildKitArgs, "buildkit-args", "", "buildkit build image container args config,need '&' split")
	fs.BoolVar(&a.BuildKitCache, "buildkit-cache", false, "whether to enable the buildkit image cache")
	fs.IntVar(&a.KeepCount, "keep-count", 5, "default number of reserved copies for images")
	fs.IntVar(&a.CleanInterval, "clean-interval", 60, "clean image interval,default 60 minute")
	fs.StringVar(&a.BRVersion, "br-version", "v5.16.0-release", "builder and runner version")

	fs.StringSliceVar(&a.EventLogServers, "event-servers", []string{"rbd-api-api-inner:6366"}, "event log server address. simple lb")
	fs.StringVar(&a.MQAPI, "mq-api", "rbd-mq:6300", "acp_mq api")

	fs.StringVar(&a.ElasticSearchURL, "es-url", "http://47.92.106.114:9200", "es url")
	fs.StringVar(&a.ElasticSearchUsername, "es-username", "", "es username")
	fs.StringVar(&a.ElasticSearchPassword, "es-password", "", "es pwd")
	fs.BoolVar(&a.ElasticEnable, "es-enable", false, "enable es")

}

// SetLog 设置log
func (a *Builder) SetLog() {
	level, err := logrus.ParseLevel(a.LogLevel)
	if err != nil {
		fmt.Println("set log level error." + err.Error())
		return
	}
	logrus.SetLevel(level)
}

// CheckConfig check config
func (a *Builder) CheckConfig() error {
	if a.Topic != client.BuilderTopic && a.Topic != client.WindowsBuilderTopic {
		return fmt.Errorf("Topic is only suppory `%s` and `%s`", client.BuilderTopic, client.WindowsBuilderTopic)
	}
	if runtime.GOOS == "windows" {
		a.Topic = "windows_builder"
	}
	return nil
}
