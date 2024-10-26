//go:build windows

/*
Copyright 2021 Loggie Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wineventlog

import (
	"strings"

	"github.com/pkg/errors"
)

// 定义默认的 Windows 事件日志类型
var defaultEventLogs = []string{"Application", "System", "Security", "Setup"}

// Config 定义 wineventlog 插件的配置结构
type Config struct {
	EventLogName string `yaml:"eventLogName"`               // 采集的 Windows 事件日志名称列表
	EventsTotal  int64  `yaml:"eventsTotal" default:"0"`    // 要采集的总事件数，<=0 时表示无限
	ByteReadSize int    `yaml:"byteReadSize" default:"100"` // 每次读取系统日志的数据条数
	Qps          int    `yaml:"qps" default:"100"`          // 每秒采集事件的数量限制
}

func (c *Config) Validate() error {
	if !stringInSlice(c.EventLogName, defaultEventLogs) {
		return errors.New("eventLogName must be one of: " + strings.Join(defaultEventLogs, ", "))
	}
	return nil
}

func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
