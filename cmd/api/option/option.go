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
	"github.com/goodrain/rainbond/api/eventlog/conf"
	utils "github.com/goodrain/rainbond/util"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// Config config
type Config struct {
	DBType                 string
	APIAddr                string
	APIHealthzAddr         string
	APIAddrSSL             string
	DBConnectionInfo       string
	EventLogServers        []string
	EventLogEndpoints      []string
	NodeAPI                []string
	BuilderAPI             []string
	V1API                  string
	MQAPI                  string
	APISSL                 bool
	APICertFile            string
	APIKeyFile             string
	APICaFile              string
	WebsocketSSL           bool
	WebsocketCertFile      string
	WebsocketKeyFile       string
	WebsocketAddr          string
	Opentsdb               string
	RegionTag              string
	LoggerFile             string
	EnableFeature          []string
	Debug                  bool
	MinExtPort             int // minimum external port
	LicensePath            string
	LicSoPath              string
	LogPath                string
	KuberentesDashboardAPI string
	KubeConfigPath         string
	PrometheusEndpoint     string
	RbdNamespace           string
	ShowSQL                bool
	GrctlImage             string
	RbdHub                 string
	RbdWorker              string
	RegionName             string
	RegionSN               string

	ElasticSearchURL      string
	ElasticSearchUsername string
	ElasticSearchPassword string
	ElasticEnable         bool
}

type EventLogConfig struct {
	Conf conf.Conf
}

// APIServer  apiserver server
type APIServer struct {
	Config
	EventLogConfig EventLogConfig
	LogLevel       string
	StartRegionAPI bool
}

// NewAPIServer new server
func NewAPIServer() *APIServer {
	return &APIServer{}
}

// AddFlags config
func (a *APIServer) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.LogLevel, "log-level", "info", "the api log level")
	fs.StringVar(&a.DBType, "db-type", "mysql", "db type mysql or etcd")
	fs.StringVar(&a.DBConnectionInfo, "mysql", "admin:admin@tcp(127.0.0.1:3306)/region", "mysql db connection info")
	fs.StringVar(&a.APIAddr, "api-addr", "0.0.0.0:8888", "the api server listen address")
	fs.StringVar(&a.APIHealthzAddr, "api-healthz-addr", "0.0.0.0:8889", "the api server health check listen address")
	fs.StringVar(&a.APIAddrSSL, "api-addr-ssl", "0.0.0.0:8443", "the api server listen address")
	fs.StringVar(&a.WebsocketAddr, "ws-addr", "0.0.0.0:6060", "the websocket server listen address")
	fs.BoolVar(&a.APISSL, "api-ssl-enable", false, "whether to enable websocket  SSL")
	fs.StringVar(&a.APICaFile, "client-ca-file", "", "api ssl ca file")
	fs.StringVar(&a.APICertFile, "api-ssl-certfile", "", "api ssl cert file")
	fs.StringVar(&a.APIKeyFile, "api-ssl-keyfile", "", "api ssl cert file")
	fs.BoolVar(&a.WebsocketSSL, "ws-ssl-enable", false, "whether to enable websocket  SSL")
	fs.StringVar(&a.WebsocketCertFile, "ws-ssl-certfile", "/etc/ssl/goodrain.com/goodrain.com.crt", "websocket and fileserver ssl cert file")
	fs.StringVar(&a.WebsocketKeyFile, "ws-ssl-keyfile", "/etc/ssl/goodrain.com/goodrain.com.key", "websocket and fileserver ssl key file")
	fs.StringVar(&a.V1API, "v1-api", "127.0.0.1:8887", "the region v1 api")
	fs.StringSliceVar(&a.BuilderAPI, "builder-api", []string{"rbd-chaos:3228"}, "the builder api")
	fs.BoolVar(&a.StartRegionAPI, "start", false, "Whether to start region old api")
	fs.StringVar(&a.Opentsdb, "opentsdb", "127.0.0.1:4242", "opentsdb server config")
	fs.StringVar(&a.RegionTag, "region-tag", "test-ali", "region tag setting")
	fs.StringVar(&a.LoggerFile, "logger-file", "/logs/request.log", "request log file path")
	fs.BoolVar(&a.Debug, "debug", false, "open debug will enable pprof")
	fs.IntVar(&a.MinExtPort, "min-ext-port", 0, "minimum external port")
	fs.StringArrayVar(&a.EnableFeature, "enable-feature", []string{}, "List of special features supported, such as `windows`")
	fs.StringVar(&a.LicensePath, "license-path", "/opt/rainbond/etc/license/license.yb", "the license path of the enterprise version.")
	fs.StringVar(&a.LicSoPath, "license-so-path", "/opt/rainbond/etc/license/license.so", "Dynamic library file path for parsing the license.")
	fs.StringVar(&a.LogPath, "log-path", "/grdata/logs", "Where Docker log files and event log files are stored.")
	fs.StringVar(&a.KubeConfigPath, "kube-config", "", "kube config file path, No setup is required to run in a cluster.")
	fs.StringVar(&a.KuberentesDashboardAPI, "k8s-dashboard-api", "kubernetes-dashboard."+utils.GetenvDefault("RBD_NAMESPACE", constants.Namespace)+":443", "The service DNS name of Kubernetes dashboard. Default to kubernetes-dashboard.kubernetes-dashboard")
	fs.StringVar(&a.RbdNamespace, "rbd-namespace", utils.GetenvDefault("RBD_NAMESPACE", constants.Namespace), "rbd component namespace")
	fs.BoolVar(&a.ShowSQL, "show-sql", false, "The trigger for showing sql.")
	fs.StringVar(&a.GrctlImage, "shell-image", "registry.cn-hangzhou.aliyuncs.com/goodrain/rbd-shell:v5.13.0-release", "use shell image")

	fs.StringVar(&a.PrometheusEndpoint, "prom-api", "rbd-monitor:9999", "The service DNS name of Prometheus api. Default to rbd-monitor:9999")
	fs.StringVar(&a.RbdHub, "hub-api", "http://rbd-hub:5000", "the rbd-hub server api")
	fs.StringSliceVar(&a.NodeAPI, "node-api", []string{"rbd-node:6100"}, "the rbd-node server api")
	fs.StringVar(&a.MQAPI, "mq-api", "rbd-mq:6300", "the rbd-mq server api")
	fs.StringVar(&a.RbdWorker, "worker-api", "rbd-worker:6535", "the rbd-worker server api")
	fs.StringSliceVar(&a.EventLogServers, "event-servers", []string{"rbd-api-api-inner:6366"}, "event log server address")
	fs.StringSliceVar(&a.EventLogEndpoints, "event-log", []string{"local=>rbd-eventlog:6363"}, "event log websocket address")

	fs.StringVar(&a.ElasticSearchURL, "es-url", "http://47.92.106.114:9200", "es url")
	fs.StringVar(&a.ElasticSearchUsername, "es-username", "", "es username")
	fs.StringVar(&a.ElasticSearchPassword, "es-password", "", "es pwd")
	fs.BoolVar(&a.ElasticEnable, "es-enable", false, "enable es")

	// event log conf
	fs.StringVar(&a.EventLogConfig.Conf.Entry.EventLogServer.BindIP, "eventlog.bind.ip", "0.0.0.0", "Collect the log service to listen the IP")
	fs.IntVar(&a.EventLogConfig.Conf.Entry.EventLogServer.BindPort, "eventlog.bind.port", 6366, "Collect the log service to listen the Port")
	fs.IntVar(&a.EventLogConfig.Conf.Entry.EventLogServer.CacheMessageSize, "eventlog.cache", 100, "the event log server cache the receive message size")
	fs.StringVar(&a.EventLogConfig.Conf.Entry.DockerLogServer.BindIP, "dockerlog.bind.ip", "0.0.0.0", "Collect the log service to listen the IP")
	fs.StringVar(&a.EventLogConfig.Conf.Entry.DockerLogServer.Mode, "dockerlog.mode", "stream", "the server mode zmq or stream")
	fs.IntVar(&a.EventLogConfig.Conf.Entry.DockerLogServer.BindPort, "dockerlog.bind.port", 6362, "Collect the log service to listen the Port")
	fs.IntVar(&a.EventLogConfig.Conf.Entry.DockerLogServer.CacheMessageSize, "dockerlog.cache", 200, "the docker log server cache the receive message size")
	fs.StringSliceVar(&a.EventLogConfig.Conf.Entry.MonitorMessageServer.SubAddress, "monitor.subaddress", []string{"tcp://127.0.0.1:9442"}, "monitor message source address")
	fs.IntVar(&a.EventLogConfig.Conf.Entry.MonitorMessageServer.CacheMessageSize, "monitor.cache", 200, "the monitor sub server cache the receive message size")
	fs.StringVar(&a.EventLogConfig.Conf.Entry.MonitorMessageServer.SubSubscribe, "monitor.subscribe", "ceptop", "the monitor message sub server subscribe info")
	fs.StringVar(&a.EventLogConfig.Conf.Cluster.Discover.InstanceIP, "cluster.instance.ip", "", "The current instance IP in the cluster can be communications.")
	fs.StringVar(&a.EventLogConfig.Conf.Cluster.Discover.Type, "discover.type", "etcd", "the instance in cluster auto discover way.")
	fs.StringVar(&a.EventLogConfig.Conf.Cluster.Discover.HomePath, "discover.etcd.homepath", "/event", "etcd home key")
	fs.StringVar(&a.EventLogConfig.Conf.Cluster.PubSub.PubBindIP, "cluster.bind.ip", "0.0.0.0", "Cluster communication to listen the IP")
	fs.IntVar(&a.EventLogConfig.Conf.Cluster.PubSub.PubBindPort, "cluster.bind.port", 6365, "Cluster communication to listen the Port")
	fs.StringVar(&a.EventLogConfig.Conf.EventStore.MessageType, "message.type", "json", "Receive and transmit the log message type.")
	fs.StringVar(&a.EventLogConfig.Conf.EventStore.GarbageMessageSaveType, "message.garbage.save", "file", "garbage message way of storage")
	fs.StringVar(&a.EventLogConfig.Conf.EventStore.GarbageMessageFile, "message.garbage.file", "/var/log/envent_garbage_message.log", "save garbage message file path when save type is file")
	fs.Int64Var(&a.EventLogConfig.Conf.EventStore.PeerEventMaxLogNumber, "message.max.number", 100000, "the max number log message for peer event")
	fs.IntVar(&a.EventLogConfig.Conf.EventStore.PeerEventMaxCacheLogNumber, "message.cache.number", 256, "Maintain log the largest number in the memory peer event")
	fs.Int64Var(&a.EventLogConfig.Conf.EventStore.PeerDockerMaxCacheLogNumber, "dockermessage.cache.number", 128, "Maintain log the largest number in the memory peer docker service")
	fs.IntVar(&a.EventLogConfig.Conf.EventStore.HandleMessageCoreNumber, "message.handle.core.number", 2, "The number of concurrent processing receive log data.")
	fs.IntVar(&a.EventLogConfig.Conf.EventStore.HandleSubMessageCoreNumber, "message.sub.handle.core.number", 3, "The number of concurrent processing receive log data. more than message.handle.core.number")
	fs.IntVar(&a.EventLogConfig.Conf.EventStore.HandleDockerLogCoreNumber, "message.dockerlog.handle.core.number", 2, "The number of concurrent processing receive log data. more than message.handle.core.number")
	fs.StringVar(&a.EventLogConfig.Conf.Log.LogLevel, "log.level", "info", "app log level")
	fs.StringVar(&a.EventLogConfig.Conf.Log.LogOutType, "log.type", "stdout", "app log output type. stdout or file ")
	fs.StringVar(&a.EventLogConfig.Conf.Log.LogPath, "log.path", "/var/log/", "app log output file path.it is effective when log.type=file")
	fs.StringVar(&a.EventLogConfig.Conf.WebSocket.BindIP, "websocket.bind.ip", "0.0.0.0", "the bind ip of websocket for push event message")
	fs.IntVar(&a.EventLogConfig.Conf.WebSocket.BindPort, "websocket.bind.port", 6363, "the bind port of websocket for push event message")
	fs.IntVar(&a.EventLogConfig.Conf.WebSocket.SSLBindPort, "websocket.ssl.bind.port", 6364, "the ssl bind port of websocket for push event message")
	fs.BoolVar(&a.EventLogConfig.Conf.WebSocket.EnableCompression, "websocket.compression", true, "weither enable compression for web socket")
	fs.IntVar(&a.EventLogConfig.Conf.WebSocket.ReadBufferSize, "websocket.readbuffersize", 4096, "the readbuffersize of websocket for push event message")
	fs.IntVar(&a.EventLogConfig.Conf.WebSocket.WriteBufferSize, "websocket.writebuffersize", 4096, "the writebuffersize of websocket for push event message")
	fs.IntVar(&a.EventLogConfig.Conf.WebSocket.MaxRestartCount, "websocket.maxrestart", 5, "the max restart count of websocket for push event message")
	fs.BoolVar(&a.EventLogConfig.Conf.WebSocket.SSL, "websocket.ssl", false, "whether to enable websocket  SSL")
	fs.StringVar(&a.EventLogConfig.Conf.WebSocket.CertFile, "websocket.certfile", "/etc/ssl/goodrain.com/goodrain.com.crt", "websocket ssl cert file")
	fs.StringVar(&a.EventLogConfig.Conf.WebSocket.KeyFile, "websocket.keyfile", "/etc/ssl/goodrain.com/goodrain.com.key", "websocket ssl cert file")
	fs.StringVar(&a.EventLogConfig.Conf.WebSocket.TimeOut, "websocket.timeout", "1m", "Keep websocket service the longest time when without message ")
	fs.StringVar(&a.EventLogConfig.Conf.WebSocket.PrometheusMetricPath, "monitor-path", "/metrics", "promethesu monitor metrics path")
	fs.StringVar(&a.EventLogConfig.Conf.EventStore.DB.Type, "db.type", "mysql", "Data persistence type.")
	fs.StringVar(&a.EventLogConfig.Conf.EventStore.DB.URL, "db.url", "root:admin@tcp(127.0.0.1:3306)/event", "Data persistence db url.")
	fs.IntVar(&a.EventLogConfig.Conf.EventStore.DB.PoolSize, "db.pool.size", 3, "Data persistence db pool init size.")
	fs.IntVar(&a.EventLogConfig.Conf.EventStore.DB.PoolMaxSize, "db.pool.maxsize", 10, "Data persistence db pool max size.")
	fs.StringVar(&a.EventLogConfig.Conf.EventStore.DB.HomePath, "docker.log.homepath", "/grdata/logs/", "container log persistent home path")
	fs.StringVar(&a.EventLogConfig.Conf.Entry.NewMonitorMessageServerConf.ListenerHost, "monitor.udp.host", "0.0.0.0", "receive new monitor udp server host")
	fs.IntVar(&a.EventLogConfig.Conf.Entry.NewMonitorMessageServerConf.ListenerPort, "monitor.udp.port", 6166, "receive new monitor udp server port")
	fs.StringVar(&a.EventLogConfig.Conf.Cluster.Discover.NodeID, "node-id", "", "the unique ID for this node.")
	fs.DurationVar(&a.EventLogConfig.Conf.Cluster.PubSub.PollingTimeout, "zmq4-polling-timeout", 200*time.Millisecond, "The timeout determines the time-out on the polling of sockets")

	fs.StringVar(&a.EventLogConfig.Conf.ElasticSearchURL, "es-url", "http://47.92.106.114:9200", "es url")
	fs.StringVar(&a.EventLogConfig.Conf.ElasticSearchUsername, "es-username", "", "es username")
	fs.StringVar(&a.EventLogConfig.Conf.ElasticSearchPassword, "es-password", "", "es pwd")
	fs.BoolVar(&a.EventLogConfig.Conf.ElasticEnable, "es-enable", false, "enable es")
}

// SetLog 设置log
func (a *APIServer) SetLog() {
	level, err := logrus.ParseLevel(a.LogLevel)
	if err != nil {
		fmt.Println("set log level error." + err.Error())
		return
	}
	logrus.SetLevel(level)
}
