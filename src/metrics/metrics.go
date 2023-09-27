package metrics

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/interfaces"
	"github.com/yuhaohwang/bililive-go/src/listeners"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/recorders"
)

// 定义一些 Prometheus 指标的描述符
var (
	liveStatus = prometheus.NewDesc(
		// 定义 liveStatus 指标的描述符
		prometheus.BuildFQName("bgo", "live", "status"),
		"live status",
		[]string{"live_id", "live_url", "live_host_name", "live_room_name", "live_listening"},
		nil,
	)
	liveDurationSeconds = prometheus.NewDesc(
		// 定义 liveDurationSeconds 指标的描述符
		prometheus.BuildFQName("bgo", "live", "duration_seconds"),
		"live status",
		[]string{"live_id", "live_url", "live_host_name", "live_room_name", "start_time"},
		nil,
	)
	recorderTotalBytes = prometheus.NewDesc(
		// 定义 recorderTotalBytes 指标的描述符
		prometheus.BuildFQName("bgo", "recorder", "total_bytes"),
		"recorder total bytes",
		[]string{"live_id", "live_url", "live_host_name", "live_room_name"},
		nil,
	)
)

// collector 结构表示 Prometheus 指标收集器
type collector struct {
	inst *instance.Instance
}

// NewCollector 创建一个新的收集器实例
func NewCollector(ctx context.Context) interfaces.Module {
	return &collector{
		inst: instance.GetInstance(ctx),
	}
}

// bool2float64 将布尔值转换为浮点数（0 或 1）
func bool2float64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// Collect 收集 Prometheus 指标
func (c collector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	for id, l := range c.inst.Lives {
		wg.Add(1)
		go func(id live.ID, l live.Live) {
			defer wg.Done()
			obj, err := c.inst.Cache.Get(l)
			if err != nil {
				return
			}
			info := obj.(*live.Info)
			listening := c.inst.ListenerManager.(listeners.Manager).HasListener(context.Background(), id)
			ch <- prometheus.MustNewConstMetric(
				liveStatus, prometheus.GaugeValue, bool2float64(info.Status),
				string(id), l.GetRawUrl(), info.HostName, info.RoomName, fmt.Sprintf("%v", listening),
			)

			if info.Status && listening {
				startTime := info.Live.GetLastStartTime()
				duration := time.Since(startTime).Seconds()

				ch <- prometheus.MustNewConstMetric(
					liveDurationSeconds, prometheus.CounterValue, duration,
					string(id), l.GetRawUrl(), info.HostName, info.RoomName, strconv.FormatInt(startTime.Unix(), 10),
				)

				if r, err := c.inst.RecorderManager.(recorders.Manager).GetRecorder(context.Background(), id); err == nil {
					if status, err := r.GetStatus(); err == nil {
						if value, err := strconv.ParseFloat(status["total_size"], 64); err == nil {
							ch <- prometheus.MustNewConstMetric(recorderTotalBytes, prometheus.CounterValue, value,
								string(id), l.GetRawUrl(), info.HostName, info.RoomName)
						}
					}
				}
			}
		}(id, l)
	}
	wg.Wait()
}

// Describe 描述 Prometheus 指标
func (collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- liveStatus
	ch <- liveDurationSeconds
	ch <- recorderTotalBytes
}

// Start 启动收集器
func (c *collector) Start(_ context.Context) error {
	return prometheus.Register(c)
}

// Close 关闭收集器
func (c *collector) Close(_ context.Context) {}
