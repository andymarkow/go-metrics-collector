package v1

import (
	"errors"
	"fmt"
	"io"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
	"github.com/andymarkow/go-metrics-collector/internal/models"
)

func (s *MetricService) UpdateMetrics(stream pbv1.MetricService_UpdateMetricServer) error {
	var metrics []models.Metrics
	var batchSize = 100

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			err := stream.SendAndClose(&pbv1.UpdateMetricResponse{
				Status: &pbv1.Status{
					Status: &wrapperspb.StringValue{Value: "OK"},
				},
			})
			if err != nil {
				s.log.Error("stream.SendAndClose", zap.Error(err))
			}

			break
		}
		if err != nil {
			return fmt.Errorf("stream.Recv: %w", err)
		}

		// Process metric request.
		metric, err := s.processUpdateMetricRequest(req)
		if err != nil {
			return fmt.Errorf("failed to process UpdateMetricRequest: %w", err)
		}

		metrics = append(metrics, metric)

		// Check if batch size limit is reached.
		if len(metrics) >= batchSize {
			// Write metrics to the storage.
			if err := s.storage.SetMetrics(stream.Context(), metrics); err != nil {
				s.log.Error("failed to write metrics to storage", zap.Error(fmt.Errorf("storage.SetMetrics: %w", err)))

				// Should continue to process in the next iteration.
				// Metrics won't be flushed to the storage.
				continue
			}

			// Flush already written to the storage metrics.
			metrics = metrics[:0]
		}
	}

	// Check if there are any remaining metrics.
	if len(metrics) > 0 {
		// Write remaining metrics to the storage.
		err := s.storage.SetMetrics(stream.Context(), metrics)
		if err != nil {
			return fmt.Errorf("storage.SetMetrics: %w", err)
		}
	}

	return nil
}

func (s *MetricService) processUpdateMetricRequest(req *pbv1.UpdateMetricRequest) (models.Metrics, error) {
	delta := req.GetDelta().GetDelta().GetValue()
	value := req.GetValue().GetValue().GetValue()

	metric, err := models.NewMetrics(
		req.GetId().GetId().GetValue(),
		req.GetMtype().GetMtype().String(),
		&delta,
		&value,
	)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("models.NewMetrics: %w", err)
	}

	return metric, nil
}
