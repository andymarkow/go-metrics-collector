package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
	"github.com/andymarkow/go-metrics-collector/internal/models"
)

// UpdateMetrics handles a gRPC UpdateMetrics request.
func (s *MetricService) UpdateMetrics(ctx context.Context, req *pbv1.UpdateMetricsRequest) (*pbv1.UpdateMetricsResponse, error) {
	var response pbv1.UpdateMetricsResponse

	ms, err := s.processUpdateMetricRequest(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process request: %v", err)
	}

	if err := s.storage.SetMetrics(ctx, ms); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to write metrics to storage: %v", err)
	}

	response.Status = &pbv1.Status{Msg: &wrapperspb.StringValue{Value: "OK"}}

	return &response, nil
}

// processUpdateMetricRequest processes a gRPC UpdateMetric request.
func (s *MetricService) processUpdateMetricRequest(req *pbv1.UpdateMetricsRequest) ([]models.Metrics, error) {
	data := req.GetPayload().GetData().GetValue()

	metricBatch, err := models.UnmarshalMetricsJSON(data)
	if err != nil {
		return nil, fmt.Errorf("models.UnmarshalMetricsJSON: %w", err)
	}

	return metricBatch, nil
}
