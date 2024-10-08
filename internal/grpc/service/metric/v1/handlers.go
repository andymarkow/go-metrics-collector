package v1

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	pbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
)

// UpdateMetrics handles a gRPC UpdateMetrics request.
func (s *MetricService) UpdateMetrics(ctx context.Context, req *pbv1.UpdateMetricsRequest) (*pbv1.UpdateMetricsResponse, error) {
	var response pbv1.UpdateMetricsResponse

	ms, err := s.processUpdateMetricRequest(ctx, req)
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
func (s *MetricService) processUpdateMetricRequest(ctx context.Context, req *pbv1.UpdateMetricsRequest) ([]models.Metrics, error) {
	data := req.GetPayload().GetData().GetValue()

	if len(s.signKey) > 0 {
		var hashsum string

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get("hashsum")

			if len(values) > 0 {
				hashsum = values[0]
			}
		}

		mdHash, err := hex.DecodeString(hashsum)
		if err != nil {
			return nil, fmt.Errorf("hex.DecodeString: %w", err)
		}

		s.log.Debug("metadata payload signature", zap.String("hashsum", hashsum))

		dataHash, err := signature.CalculateHashSum(s.signKey, data)
		if err != nil {
			return nil, fmt.Errorf("signature.CalculateHashSum: %w", err)
		}

		s.log.Debug("data payload signature", zap.String("hashsum", hex.EncodeToString(dataHash)))

		if !hmac.Equal(mdHash, dataHash) {
			s.log.Error("signature mismatch", zap.Error(errormsg.ErrHashSumValueMismatch))

			return nil, fmt.Errorf("signature mismatch: %w", errormsg.ErrHashSumValueMismatch)
		}
	}

	metricBatch, err := models.UnmarshalMetricsJSON(data)
	if err != nil {
		return nil, fmt.Errorf("models.UnmarshalMetricsJSON: %w", err)
	}

	return metricBatch, nil
}
