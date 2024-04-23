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

package controller

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond/api/util"
	dbmodel "github.com/goodrain/rainbond/db/model"
	httputil "github.com/goodrain/rainbond/util/http"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi"
	"github.com/goodrain/rainbond/api/discover"
	"github.com/goodrain/rainbond/api/handler"
	"github.com/goodrain/rainbond/api/proxy"
	ctxutil "github.com/goodrain/rainbond/api/util/ctx"
	"github.com/goodrain/rainbond/util/constants"
	"github.com/sirupsen/logrus"
)

// DockerConsole docker console
type DockerConsole struct {
	socketproxy proxy.Proxy
}

var defaultDockerConsoleEndpoints = []string{"127.0.0.1:7171"}
var defaultEventLogEndpoints = []string{"local=>rbd-eventlog:6363"}

var dockerConsole *DockerConsole

// GetDockerConsole get Docker console
func GetDockerConsole() *DockerConsole {
	if dockerConsole != nil {
		return dockerConsole
	}
	dockerConsole = &DockerConsole{
		socketproxy: proxy.CreateProxy("dockerconsole", "websocket", defaultDockerConsoleEndpoints),
	}
	discover.GetEndpointDiscover().AddProject("acp_webcli", dockerConsole.socketproxy)
	return dockerConsole
}

// Get get
func (d DockerConsole) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

var dockerLog *DockerLog

// DockerLog docker log
type DockerLog struct {
	socketproxy proxy.Proxy
}

// GetDockerLog get docker log
func GetDockerLog() *DockerLog {
	if dockerLog == nil {
		dockerLog = &DockerLog{
			socketproxy: proxy.CreateProxy("dockerlog", "websocket", defaultEventLogEndpoints),
		}
		discover.GetEndpointDiscover().AddProject("event_log_event_http", dockerLog.socketproxy)
	}
	return dockerLog
}

// Get get
func (d DockerLog) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

// MonitorMessage monitor message
type MonitorMessage struct {
	socketproxy proxy.Proxy
}

var monitorMessage *MonitorMessage

// GetMonitorMessage get MonitorMessage
func GetMonitorMessage() *MonitorMessage {
	if monitorMessage == nil {
		monitorMessage = &MonitorMessage{
			socketproxy: proxy.CreateProxy("monitormessage", "websocket", defaultEventLogEndpoints),
		}
		discover.GetEndpointDiscover().AddProject("event_log_event_http", monitorMessage.socketproxy)
	}
	return monitorMessage
}

// Get get
func (d MonitorMessage) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

// EventLog event log
type EventLog struct {
	socketproxy proxy.Proxy
}

var eventLog *EventLog

// GetEventLog get event log
func GetEventLog() *EventLog {
	if eventLog == nil {
		eventLog = &EventLog{
			socketproxy: proxy.CreateProxy("eventlog", "websocket", defaultEventLogEndpoints),
		}
		discover.GetEndpointDiscover().AddProject("event_log_event_http", eventLog.socketproxy)
	}
	return eventLog
}

// Get get
func (d EventLog) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

// LogFile log file down server
type LogFile struct {
	Root string
}

var logFile *LogFile

// GetLogFile get  log file
func GetLogFile() *LogFile {
	root := os.Getenv("SERVICE_LOG_ROOT")
	if root == "" {
		root = constants.GrdataLogPath
	}
	logrus.Infof("service logs file root path is :%s", root)
	if logFile == nil {
		logFile = &LogFile{
			Root: root,
		}
	}
	return logFile
}

// Get get
func (d LogFile) Get(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "gid")
	filename := chi.URLParam(r, "filename")
	filePath := path.Join(d.Root, gid, filename)
	if isExist(filePath) {
		http.ServeFile(w, r, filePath)
	} else {
		w.WriteHeader(404)
	}
}
func isExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// GetInstallLog get
func (d LogFile) GetInstallLog(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	filePath := d.Root + filename
	if isExist(filePath) {
		http.ServeFile(w, r, filePath)
	} else {
		w.WriteHeader(404)
	}
}

var pubSubControll *PubSubControll

// PubSubControll service pub sub
type PubSubControll struct {
	socketproxy proxy.Proxy
}

// GetPubSubControll get service pub sub controller
func GetPubSubControll() *PubSubControll {
	if pubSubControll == nil {
		pubSubControll = &PubSubControll{
			socketproxy: proxy.CreateProxy("dockerlog", "websocket", defaultEventLogEndpoints),
		}
		discover.GetEndpointDiscover().AddProject("event_log_event_http", pubSubControll.socketproxy)
	}
	return pubSubControll
}

// Get pubsub controller
func (d PubSubControll) Get(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "serviceID")
	name, _ := handler.GetEventHandler().GetLogInstance(serviceID)
	if name != "" {
		r.URL.Query().Add("host_id", name)
		r = r.WithContext(context.WithValue(r.Context(), proxy.ContextKey("host_id"), name))
	}
	d.socketproxy.Proxy(w, r)
}

// GetHistoryLog get service docker logs
func (d PubSubControll) GetHistoryLog(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	name, _ := handler.GetEventHandler().GetLogInstance(serviceID)
	if name != "" {
		r.URL.Query().Add("host_id", name)
		r = r.WithContext(context.WithValue(r.Context(), proxy.ContextKey("host_id"), name))
	}
	d.socketproxy.Proxy(w, r)
}

var fileManage *FileManage

// FileManage docker log
type FileManage struct {
	socketproxy proxy.Proxy
}

// GetFileManage get docker log
func GetFileManage() *FileManage {
	if fileManage == nil {
		fileManage = &FileManage{
			socketproxy: proxy.CreateProxy("acp_node", "http", []string{"127.0.0.1:6100"}),
		}
		discover.GetEndpointDiscover().AddProject("acp_node", fileManage.socketproxy)
	}
	return fileManage
}

// Get get
func (f FileManage) Get(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Add("Access-Control-Allow-Origin", origin)
	w.Header().Add("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "x-requested-with,Content-Type,X-Custom-Header")
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/octet-stream")
		f.DownloadFile(w, r)
	case "POST":
		f.UploadFile(w, r)
		f.UploadEvent(w, r)
	case "OPTIONS":
		httputil.ReturnSuccess(r, w, nil)
	}
}

// UploadEvent volume file upload event
func (f FileManage) UploadEvent(w http.ResponseWriter, r *http.Request) {
	volumeName := w.Header().Get("volume_name")
	userName := w.Header().Get("user_name")
	tenantID := w.Header().Get("tenant_id")
	serviceID := w.Header().Get("service_id")
	fileName := w.Header().Get("file_name")
	status := w.Header().Get("status")
	msg := fmt.Sprintf("%v to upload file %v in storage %v", status, fileName, volumeName)
	_, err := util.CreateEvent(dbmodel.TargetTypeService, "volume-file-upload", serviceID, tenantID, "", userName, status, msg, 1)
	if err != nil {
		logrus.Error("create event error: ", err)
		httputil.ReturnError(r, w, 500, "操作失败")
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (f FileManage) UploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("volume_name", r.FormValue("volume_name"))
	w.Header().Add("user_name", r.FormValue("user_name"))
	w.Header().Add("tenant_id", r.FormValue("tenant_id"))
	w.Header().Add("service_id", r.FormValue("service_id"))
	w.Header().Add("status", "failed")
	destPath := r.FormValue("path")
	podName := r.FormValue("pod_name")
	namespace := r.FormValue("namespace")
	containerName := r.FormValue("container_name")
	if destPath == "" {
		httputil.ReturnError(r, w, 400, "Path cannot be empty")
		return
	}
	reader, header, err := r.FormFile("file")
	if err != nil {
		logrus.Errorf("Failed to parse upload file: %s", err.Error())
		httputil.ReturnError(r, w, 501, "Failed to parse upload file.")
		return
	}
	defer reader.Close()
	w.Header().Add("file_name", header.Filename)
	srcPath := path.Join("./", header.Filename)
	file, err := os.OpenFile(srcPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("upload file open %v failure: %v", header.Filename, err.Error())
		httputil.ReturnError(r, w, 502, "Failed to open file: "+err.Error())
	}
	defer os.Remove(srcPath)
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		logrus.Errorf("upload file write %v failure: %v", srcPath, err.Error())
		httputil.ReturnError(r, w, 503, "Failed to write file: "+err.Error())
		return
	}
	err = handler.GetServiceManager().AppFileUpload(containerName, podName, srcPath, destPath, namespace)
	if err != nil {
		logrus.Errorf("upload file %v to %v %v failure: %v", header.Filename, podName, destPath, err.Error())
		httputil.ReturnError(r, w, 503, "Failed to write file: "+err.Error())
		return
	}
	w.Header().Set("status", "success")
}

func (f FileManage) DownloadFile(w http.ResponseWriter, r *http.Request) {
	podName := r.FormValue("pod_name")
	p := r.FormValue("path")
	namespace := r.FormValue("namespace")
	fileName := strings.TrimSpace(chi.URLParam(r, "fileName"))

	filePath := path.Join(p, fileName)
	containerName := r.FormValue("container_name")

	err := handler.GetServiceManager().AppFileDownload(containerName, podName, filePath, namespace)
	if err != nil {
		logrus.Errorf("downloading file from Pod failure: %v", err)
		http.Error(w, "Error downloading file from Pod", http.StatusInternalServerError)
		return
	}
	defer os.Remove(path.Join("./", fileName))
	w.Header().Set("Content-Disposition", "attachment;filename="+fileName)
	http.ServeFile(w, r, path.Join("./", fileName))
}
