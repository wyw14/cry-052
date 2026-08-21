package service

import "context"

func (p *BatchProcessor) settleCancellation(ctx context.Context, batchID string, cause error) error {
	// State persistence follows the request lifetime so shutdown remains prompt.
	return p.cancelled(ctx, batchID, cause)
}
