package metrics

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"ipscope/internal/geolocation"
	"ipscope/internal/model"

	"github.com/prometheus/client_golang/prometheus"
)

var invalidMetricChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type Exporter struct {
	resolver         geolocation.Resolver
	datacenterInfo   *prometheus.GaugeVec
	resolveErrorFlag *prometheus.GaugeVec
}

func NewExporter(registry prometheus.Registerer, prefix string, resolver geolocation.Resolver) (*Exporter, error) {
	normalizedPrefix := normalizeMetricPrefix(prefix)

	datacenterInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: normalizedPrefix + "_node_datacenter_info",
			Help: "Node datacenter identity labels. Gauge value is always 1.",
		},
		[]string{"node", "endpoint", "datacenter", "city", "region", "country", "latitude", "longitude"},
	)

	resolveErrorFlag := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: normalizedPrefix + "_node_datacenter_lookup_error",
			Help: "Datacenter lookup error status (1=error, 0=success).",
		},
		[]string{"node", "endpoint"},
	)

	for _, collector := range []prometheus.Collector{datacenterInfo, resolveErrorFlag} {
		if err := registry.Register(collector); err != nil {
			return nil, fmt.Errorf("register prometheus collector: %w", err)
		}
	}

	return &Exporter{
		resolver:         resolver,
		datacenterInfo:   datacenterInfo,
		resolveErrorFlag: resolveErrorFlag,
	}, nil
}

func (e *Exporter) Refresh(ctx context.Context, nodes []model.NodeConfig) error {
	var errs []error

	for _, node := range nodes {
		info, err := e.resolver.ResolveDatacenter(ctx, node.Endpoint)
		if err != nil {
			errs = append(errs, fmt.Errorf("resolve node %q (%s): %w", node.Name, node.Endpoint, err))
			info = model.DatacenterInfo{
				Datacenter: "unknown",
				City:       "unknown",
				Region:     "unknown",
				Country:    "unknown",
				Latitude:   0,
				Longitude:  0,
			}
			e.resolveErrorFlag.WithLabelValues(node.Name, node.Endpoint).Set(1)
		} else {
			e.resolveErrorFlag.WithLabelValues(node.Name, node.Endpoint).Set(0)
		}

		e.datacenterInfo.WithLabelValues(
			node.Name,
			node.Endpoint,
			info.Datacenter,
			valueOrUnknown(info.City),
			valueOrUnknown(info.Region),
			valueOrUnknown(info.Country),
			formatCoordinate(info.Latitude),
			formatCoordinate(info.Longitude),
		).Set(1)
	}

	return errors.Join(errs...)
}

func normalizeMetricPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return model.DefaultPrefix
	}

	normalized := invalidMetricChars.ReplaceAllString(trimmed, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return model.DefaultPrefix
	}

	return normalized
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}

	return value
}

func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}
