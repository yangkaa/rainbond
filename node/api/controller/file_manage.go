package controller

import (
	"fmt"
	"github.com/go-chi/chi"
	httputil "github.com/goodrain/rainbond/util/http"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
)

// UploadFile -
func UploadFile(w http.ResponseWriter, r *http.Request) {
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
	srcPath := fmt.Sprintf("./%s", header.Filename)
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
	}

	err = appService.AppFileUpload(containerName, podName, srcPath, destPath, namespace)
	if err != nil {
		logrus.Errorf("upload file %v to %v %v failure: %v", header.Filename, podName, destPath, err.Error())
		httputil.ReturnError(r, w, 503, "Failed to write file: "+err.Error())
	}
	w.Header().Set("status", "success")
}

//DownloadFile -
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	podName := r.FormValue("pod_name")
	path := r.FormValue("path")
	namespace := r.FormValue("namespace")
	fileName := strings.TrimSpace(chi.URLParam(r, "fileName"))
	filePath := fmt.Sprintf("%s/%s", path, fileName)
	containerName := r.FormValue("container_name")

	err := appService.AppFileDownload(containerName, podName, filePath, namespace)
	if err != nil {
		logrus.Errorf("downloading file from Pod failure: %v", err)
		http.Error(w, "Error downloading file from Pod", http.StatusInternalServerError)
		return
	}
	defer os.Remove(fmt.Sprintf("./%s", fileName))
	w.Header().Set("Content-Disposition", "attachment;filename="+fileName)
	http.ServeFile(w, r, fmt.Sprintf("./%s", fileName))
}
