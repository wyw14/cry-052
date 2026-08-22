package service

import "context"

// settleCancellation persists the cancelled terminal state. ctx must be detached
// from the request lifetime (callers pass a WithoutCancel context) so that an
// externally cancelled request still leaves the batch in a recoverable state
// instead of stuck in running.
func (p *BatchProcessor) settleCancellation(ctx context.Context, batchID string, cause error) error {
	return p.cancelled(ctx, batchID, cause)
}
