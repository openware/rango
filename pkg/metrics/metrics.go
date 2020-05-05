package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var defaultMetrics *Metrics

type Metrics struct {
	clients prometheus.Gauge
	subs    *prometheus.GaugeVec
}

func Enable() {
	defaultMetrics = &Metrics{}
	registerMetrics()
}

func registerMetrics() {
	defaultMetrics.clients = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "rango_hub_clients_count",
			Help: "Number of clients currently connected",
		},
	)

	defaultMetrics.subs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rango_hub_subscriptions_count",
			Help: "Number of user subscribed to a topic",
		},
		[]string{"type", "topic"},
	)
}

func RecordHubClientNew() {
	if defaultMetrics == nil {
		return
	}
	defaultMetrics.clients.Inc()
}

func RecordHubClientClose() {
	if defaultMetrics == nil {
		return
	}
	defaultMetrics.clients.Dec()
}

func RecordHubSubscription(typ, topic string) {
	if defaultMetrics == nil {
		return
	}
	defaultMetrics.subs.WithLabelValues(typ, topic).Inc()
}

func RecordHubUnsubscription(typ, topic string) {
	if defaultMetrics == nil {
		return
	}
	defaultMetrics.subs.WithLabelValues(typ, topic).Dec()
}
