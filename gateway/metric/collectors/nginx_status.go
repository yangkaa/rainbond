// RAINBOND, Application Management Platform
// Copyright (C) 2014-2019 Goodrain Co., Ltd.

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

package collectors

import (
	"fmt"
	"github.com/goodrain/rainbond/cmd/gateway/option"
	"github.com/goodrain/rainbond/gateway/controller/openresty/nginxcmd"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

var StatusPath = "/nginx_status"

var (
	ac      = regexp.MustCompile(`Active connections: (\d+)`)
	sahr    = regexp.MustCompile(`(\d+)\s(\d+)\s(\d+)`)
	reading = regexp.MustCompile(`Reading: (\d+)`)
	writing = regexp.MustCompile(`Writing: (\d+)`)
	waiting = regexp.MustCompile(`Waiting: (\d+)`)
)

type (
	basicStatus struct {
		// Active total number of active connections
		Active int
		// Accepted total number of accepted client connections
		Accepted int
		// Handled total number of handled connections. Generally, the parameter value is the same as accepts unless some resource limits have been reached (for example, the worker_connections limit).
		Handled int
		// Requests total number of client requests.
		Requests int
		// Reading current number of connections where nginx is reading the request header.
		Reading int
		// Writing current number of connections where nginx is writing the response back to the client.
		Writing int
		// Waiting current number of idle client connections waiting for a request.
		Waiting int
	}
)

func toInt(data []string, pos int) int {
	if len(data) == 0 {
		return 0
	}
	if pos > len(data) {
		return 0
	}
	if v, err := strconv.Atoi(data[pos]); err == nil {
		return v
	}
	return 0
}

func parse(data string) *basicStatus {
	acr := ac.FindStringSubmatch(data)
	sahrr := sahr.FindStringSubmatch(data)
	readingr := reading.FindStringSubmatch(data)
	writingr := writing.FindStringSubmatch(data)
	waitingr := waiting.FindStringSubmatch(data)

	return &basicStatus{
		toInt(acr, 1),
		toInt(sahrr, 1),
		toInt(sahrr, 2),
		toInt(sahrr, 3),
		toInt(readingr, 1),
		toInt(writingr, 1),
		toInt(waitingr, 1),
	}
}

// NewGetStatusRequest creates a new GET request to the internal NGINX status server
func NewGetStatusRequest(path string, listenPorts option.ListenPorts) (int, []byte, error) {
	url := fmt.Sprintf("http://127.0.0.1:%v%v", listenPorts.HTTP, path)

	client := http.Client{}
	res, err := client.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, err
	}

	return res.StatusCode, data, nil
}

//NginxCmdMetric -
type NginxCmdMetric struct {
}

//Describe -
func (n *NginxCmdMetric) Describe(ch chan<- *prometheus.Desc) {
	nginxcmd.PromethesuScrape(ch)
}

//Collect -
func (n *NginxCmdMetric) Collect(ch chan<- prometheus.Metric) {
	nginxcmd.PrometheusCollect(ch)
}
