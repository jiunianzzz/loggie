package wineventlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
	"github.com/loggie-io/loggie/pkg/core/api"
	"github.com/loggie-io/loggie/pkg/core/event"
	"github.com/loggie-io/loggie/pkg/core/log"
	"github.com/loggie-io/loggie/pkg/pipeline"
	"golang.org/x/time/rate"
)

const Type = "wineventlog"

func init() {
	pipeline.Register(api.SOURCE, Type, makeSource)
}

func makeSource(info pipeline.Info) api.Component {
	return &Source{
		stop:      make(chan struct{}),
		config:    &Config{},
		eventPool: info.EventPool,
	}
}

type Source struct {
	name      string
	stop      chan struct{}
	eventPool *event.Pool
	config    *Config
	eventLog  eventlog.EventLog
}

func (d *Source) Config() interface{} {
	return d.config
}

func (d *Source) Category() api.Category {
	return api.SOURCE
}

func (d *Source) Type() api.Type {
	return Type
}

func (d *Source) String() string {
	return fmt.Sprintf("%s/%s", api.SOURCE, Type)
}

func (d *Source) Init(context api.Context) error {
	d.name = context.Name()
	return nil
}

func (d *Source) Start() error {

	log.Info("start source: %s", d.String())
	config := common.MustNewConfigFrom(map[string]interface{}{
		"name":            d.config.EventLogName, // 读取的事件日志名称，例如 "Application", "System" 等
		"batch_read_size": d.config.ByteReadSize, // 每次读取的事件数量
	})

	eventLog, err := eventlog.New(config)
	if err != nil {
		log.Panic("Failed to create event log: ", err)
	}
	d.eventLog = eventLog
	return nil
}

func (d *Source) Stop() {
	log.Info("stopping source unix: %s", d.name)
	close(d.stop)
}

func (d *Source) ProductLoop(productFunc api.ProductFunc) {
	log.Info("%s start product loop", d.String())

	for {
		select {
		case <-d.stop:
			return
		default:
			d.collectWindowsEventLogs(d.eventLog, d.stop, d.config.EventsTotal, d.config.Qps, productFunc)
		}

	}

}

func (d *Source) Commit(events []api.Event) {
	d.eventPool.PutAll(events)
}

// collectWindowsEventLogs collects event logs from the Windows event log
func (d *Source) collectWindowsEventLogs(eventLog eventlog.EventLog, stop chan struct{}, totalCount int64, qps int, productFunc api.ProductFunc) {
	err := eventLog.Open(checkpoint.EventLogState{})
	if err != nil {
		log.Panic("Failed to open event log: %+v", err)
	}

	defer eventLog.Close()

	total := totalCount
	var index int64
	limiter := rate.NewLimiter(rate.Limit(qps), qps)

	for {
		select {
		case <-stop:
			return

		default:
			// Limit the read rate
			limiter.Wait(context.TODO())

			// Read from the event log
			records, err := eventLog.Read()
			if err != nil {
				log.Warn("Failed to read event log: ", err)
				return
			}

			for _, record := range records {
				//data, _ := json.Marshal(record)
				bf := bytes.NewBuffer([]byte{})
				encoder := json.NewEncoder(bf)
				encoder.SetEscapeHTML(false)

				encoder.Encode(record)
				e := d.eventPool.Get()
				e.Fill(e.Meta(), e.Header(), bf.Bytes())
				productFunc(e)

				index++
				if total > 0 {
					total--
					if total == 0 {
						return
					}
				}
			}
		}
	}
}
