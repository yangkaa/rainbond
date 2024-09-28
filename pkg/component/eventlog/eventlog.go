// RAINBOND, Application Management Platform
// Copyright (C) 2021-2024 Goodrain Co., Ltd.

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

package eventlog

import (
	"context"
	"github.com/goodrain/rainbond/api/eventlog/db"
	"github.com/goodrain/rainbond/api/eventlog/entry"
	"github.com/goodrain/rainbond/api/eventlog/exit/web"
	"github.com/goodrain/rainbond/api/eventlog/store"
	"github.com/goodrain/rainbond/config/configs"
	"github.com/goodrain/rainbond/pkg/component/es"
	"github.com/sirupsen/logrus"
)

var defaultEventlogComponent *EventlogComponent

// EventlogComponent -
type EventlogComponent struct {
	Entry        *entry.Entry
	SocketServer *web.SocketServer
}

// New -
func New() *EventlogComponent {
	defaultEventlogComponent = &EventlogComponent{}
	return defaultEventlogComponent
}

// Start -
func (s *EventlogComponent) Start(ctx context.Context, cfg *configs.Config) error {
	logrus.Debug("Start run server.")

	if cfg.EventlogConfig.Conf.ElasticEnable {
		es.New().SingleStart(cfg.EventlogConfig.Conf.ElasticSearchURL, cfg.EventlogConfig.Conf.ElasticSearchUsername, cfg.EventlogConfig.Conf.ElasticSearchPassword)
	}

	//init new db
	if err := db.CreateDBManager(cfg.EventlogConfig.Conf.EventStore.DB); err != nil {
		logrus.Infof("create db manager error, %v", err)
		return err
	}

	storeManager, err := store.NewManager(cfg.EventlogConfig.Conf.EventStore, logrus.WithField("module", "MessageStore"))
	if err != nil {
		return err
	}
	healthInfo := storeManager.HealthCheck()
	if err := storeManager.Run(); err != nil {
		return err
	}
	defer storeManager.Stop()

	s.SocketServer = web.NewSocket(cfg.EventlogConfig.Conf.WebSocket, cfg.EventlogConfig.Conf.Cluster.Discover,
		logrus.WithField("module", "SocketServer"), storeManager, healthInfo)
	if err := s.SocketServer.Run(); err != nil {
		return err
	}
	defer s.SocketServer.Stop()

	s.Entry = entry.NewEntry(cfg.EventlogConfig.Conf.Entry, logrus.WithField("module", "EntryServer"), storeManager)
	if err := s.Entry.Start(); err != nil {
		return err
	}
	defer s.Entry.Stop()
	return nil
}

// CloseHandle -
func (r *EventlogComponent) CloseHandle() {

}

// Default -
func Default() *EventlogComponent {
	return defaultEventlogComponent
}
