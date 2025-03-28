package directory

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ActiveSessionsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "ws2wh",
		Name:      "active_sessions",
		Help:      "The number of currently active sessions",
	})

	ConnectCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "ws2wh",
		Name:      "connects_total",
		Help:      "Connect events counter",
	})

	DisconnectCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ws2wh",
		Name:      "disconnects_total",
		Help:      "Disconnect events counter",
	}, []string{OriginLabel})

	MessageSuccessCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ws2wh",
		Name:      "message_delivered_total",
		Help:      "Successful message delivery counter",
	}, []string{OriginLabel})

	MessageFailureCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ws2wh",
		Name:      "message_failure_total",
		Help:      "Failed message delivery counter",
	}, []string{OriginLabel})
)

const (
	OriginLabel        = "origin"
	OriginValueBackend = "backend"
	OriginValueClient  = "client"
)
