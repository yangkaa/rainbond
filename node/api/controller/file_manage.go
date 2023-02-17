package controller

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/goodrain/rainbond/api/model"
	"github.com/goodrain/rainbond/util"
	httputil "github.com/goodrain/rainbond/util/http"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
)

// GetFileDir volume file manage get file and dir
func GetFileDir(w http.ResponseWriter, r *http.Request) {
	var fileInfos []model.FileInfo
	tarPath := r.FormValue("path")
	if ok := util.DirIsEmpty(tarPath); ok {
		httputil.ReturnSuccess(r, w, fileInfos)
	}
	files, err := os.ReadDir(tarPath)
	if err != nil {
		httputil.ReturnError(r, w, 400, "read dir error")
	}
	for _, file := range files {
		fileInfos = append(fileInfos, model.FileInfo{
			Title:  file.Name(),
			IsLeaf: file.IsDir(),
		})
	}
	httputil.ReturnSuccess(r, w, fileInfos)
}

// UploadFile -
func UploadFile(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	if path == "" {
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

	fileName := fmt.Sprintf("%s/%s", path, header.Filename)
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("Failed to open file: %s", err.Error())
		httputil.ReturnError(r, w, 502, "Failed to open file: "+err.Error())
	}
	defer file.Close()

	logrus.Debug("Start write file to: ", fileName)
	if _, err := io.Copy(file, reader); err != nil {
		logrus.Errorf("Failed to write file：%s", err.Error())
		httputil.ReturnError(r, w, 503, "Failed to write file: "+err.Error())
	}

	logrus.Debug("successful write file to: ", fileName)
	httputil.ReturnSuccess(r, w, nil)
}


//DownloadFile -
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	Path := r.FormValue("path")
	fileName := strings.TrimSpace(chi.URLParam(r, "fileName"))

	if Path == "" {
		httputil.ReturnError(r, w, 400, "Path cannot be empty")
		return
	}
	filePath := fmt.Sprintf("%s/%s", Path, fileName)
	// return status code 404 if the file not exists.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		httputil.ReturnError(r, w, 404, fmt.Sprintf("Not found export app tar file: %s", filePath))
		return
	}
	http.ServeFile(w, r, filePath)
}
