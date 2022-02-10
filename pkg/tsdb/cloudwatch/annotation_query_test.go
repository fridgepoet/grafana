package cloudwatch

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery_AnnotationQuery(t *testing.T) {
	origNewCWClient := NewCWClient
	t.Cleanup(func() {
		NewCWClient = origNewCWClient
	})

	var client FakeCWAnnotationsClient

	NewCWClient = func(sess *session.Session) cloudwatchiface.CloudWatchAPI {
		return client
	}

	client = FakeCWAnnotationsClient{}
	im := datasource.NewInstanceManager(func(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		return datasourceInfo{}, nil
	})

	executor := newExecutor(im, newTestConfig(), fakeSessionCache{})
	resp, err := executor.QueryData(context.Background(), &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
		Queries: []backend.DataQuery{
			{
				JSON: json.RawMessage(`{
						"type":    "annotationQuery",
						"region":    "us-east-1",
						"namespace": "custom",
						"metricName": "CPUUtilization",
						"statistic": "Average"
					}`),
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t,
		&backend.QueryDataResponse{
			Responses: backend.Responses{
				"": {
					Frames: data.Frames{data.NewFrame("",
						data.NewField("time", nil, []string{}),
						data.NewField("title", nil, []string{}),
						data.NewField("tags", nil, []string{}),
						data.NewField("text", nil, []string{}),
					).SetMeta(&data.FrameMeta{
						Custom: map[string]interface{}{
							"rowCount": 0,
						},
					})},
				},
			},
		}, resp)
}
