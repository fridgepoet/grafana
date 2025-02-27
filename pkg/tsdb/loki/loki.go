package loki

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana/pkg/infra/httpclient"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/loki/pkg/logcli/client"
	"github.com/grafana/loki/pkg/loghttp"
	"github.com/grafana/loki/pkg/logproto"
	"go.opentelemetry.io/otel/attribute"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

type Service struct {
	im     instancemgmt.InstanceManager
	plog   log.Logger
	tracer tracing.Tracer
}

func ProvideService(httpClientProvider httpclient.Provider, tracer tracing.Tracer) *Service {
	return &Service{
		im:     datasource.NewInstanceManager(newInstanceSettings(httpClientProvider)),
		plog:   log.New("tsdb.loki"),
		tracer: tracer,
	}
}

var (
	legendFormat = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
)

type datasourceInfo struct {
	HTTPClient        *http.Client
	URL               string
	TLSClientConfig   *tls.Config
	BasicAuthUser     string
	BasicAuthPassword string
	TimeInterval      string `json:"timeInterval"`
}

type QueryModel struct {
	QueryType    string `json:"queryType"`
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	Interval     string `json:"interval"`
	IntervalMS   int    `json:"intervalMS"`
	Resolution   int64  `json:"resolution"`
}

func newInstanceSettings(httpClientProvider httpclient.Provider) datasource.InstanceFactoryFunc {
	return func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		opts, err := settings.HTTPClientOptions()
		if err != nil {
			return nil, err
		}

		client, err := httpClientProvider.New(opts)
		if err != nil {
			return nil, err
		}

		tlsClientConfig, err := httpClientProvider.GetTLSConfig(opts)
		if err != nil {
			return nil, err
		}

		jsonData := datasourceInfo{}
		err = json.Unmarshal(settings.JSONData, &jsonData)
		if err != nil {
			return nil, fmt.Errorf("error reading settings: %w", err)
		}

		model := &datasourceInfo{
			HTTPClient:        client,
			URL:               settings.URL,
			TLSClientConfig:   tlsClientConfig,
			TimeInterval:      jsonData.TimeInterval,
			BasicAuthUser:     settings.BasicAuthUser,
			BasicAuthPassword: settings.DecryptedSecureJSONData["basicAuthPassword"],
		}
		return model, nil
	}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	result := backend.NewQueryDataResponse()

	dsInfo, err := s.getDSInfo(req.PluginContext)
	if err != nil {
		return result, err
	}

	client := &client.DefaultClient{
		Address:  dsInfo.URL,
		Username: dsInfo.BasicAuthUser,
		Password: dsInfo.BasicAuthPassword,
		TLSConfig: config.TLSConfig{
			InsecureSkipVerify: dsInfo.TLSClientConfig.InsecureSkipVerify,
		},
		Tripperware: func(t http.RoundTripper) http.RoundTripper {
			return dsInfo.HTTPClient.Transport
		},
	}

	queries, err := parseQuery(req)
	if err != nil {
		return result, err
	}

	for _, query := range queries {
		s.plog.Debug("Sending query", "start", query.Start, "end", query.End, "step", query.Step, "query", query.Expr)
		_, span := s.tracer.Start(ctx, "alerting.loki")
		span.SetAttributes("expr", query.Expr, attribute.Key("expr").String(query.Expr))
		span.SetAttributes("start_unixnano", query.Start, attribute.Key("start_unixnano").Int64(query.Start.UnixNano()))
		span.SetAttributes("stop_unixnano", query.End, attribute.Key("stop_unixnano").Int64(query.End.UnixNano()))
		defer span.End()

		frames, err := runQuery(client, query)

		queryRes := backend.DataResponse{}

		if err != nil {
			queryRes.Error = err
		} else {
			queryRes.Frames = frames
		}

		result.Responses[query.RefID] = queryRes
	}
	return result, nil
}

//If legend (using of name or pattern instead of time series name) is used, use that name/pattern for formatting
func formatLegend(metric model.Metric, query *lokiQuery) string {
	if query.LegendFormat == "" {
		return metric.String()
	}

	result := legendFormat.ReplaceAllFunc([]byte(query.LegendFormat), func(in []byte) []byte {
		labelName := strings.Replace(string(in), "{{", "", 1)
		labelName = strings.Replace(labelName, "}}", "", 1)
		labelName = strings.TrimSpace(labelName)
		if val, exists := metric[model.LabelName(labelName)]; exists {
			return []byte(val)
		}
		return []byte{}
	})

	return string(result)
}

func parseResponse(value *loghttp.QueryResponse, query *lokiQuery) (data.Frames, error) {
	frames := data.Frames{}

	//We are currently processing only matrix results (for alerting)
	matrix, ok := value.Data.Result.(loghttp.Matrix)
	if !ok {
		return frames, fmt.Errorf("unsupported result format: %q", value.Data.ResultType)
	}

	for _, v := range matrix {
		name := formatLegend(v.Metric, query)
		tags := make(map[string]string, len(v.Metric))
		timeVector := make([]time.Time, 0, len(v.Values))
		values := make([]float64, 0, len(v.Values))

		for k, v := range v.Metric {
			tags[string(k)] = string(v)
		}

		for _, k := range v.Values {
			timeVector = append(timeVector, k.Timestamp.Time().UTC())
			values = append(values, float64(k.Value))
		}

		timeField := data.NewField("time", nil, timeVector)
		timeField.Config = &data.FieldConfig{Interval: float64(query.Step.Milliseconds())}
		valueField := data.NewField("value", tags, values).SetConfig(&data.FieldConfig{DisplayNameFromDS: name})

		frames = append(frames, data.NewFrame(name, timeField, valueField))
	}

	return frames, nil
}

// we extracted this part of the functionality to make it easy to unit-test it
func runQuery(client *client.DefaultClient, query *lokiQuery) (data.Frames, error) {
	// `limit` only applies to log-producing queries, and we
	// currently only support metric queries, so this can be set to any value.
	limit := 1

	// we do not use `interval`, so we set it to zero
	interval := time.Duration(0)

	value, err := client.QueryRange(query.Expr, limit, query.Start, query.End, logproto.BACKWARD, query.Step, interval, false)
	if err != nil {
		return data.Frames{}, err
	}

	return parseResponse(value, query)
}

func (s *Service) getDSInfo(pluginCtx backend.PluginContext) (*datasourceInfo, error) {
	i, err := s.im.Get(pluginCtx)
	if err != nil {
		return nil, err
	}

	instance, ok := i.(*datasourceInfo)
	if !ok {
		return nil, fmt.Errorf("failed to cast datsource info")
	}

	return instance, nil
}
