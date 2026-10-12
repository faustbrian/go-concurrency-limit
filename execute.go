package concurrencylimit

import "context"

// Execute acquires a permit, invokes operation once, classifies its completion,
// and attempts one terminal completion. A rejected operation is not called.
// Work runs synchronously with the caller context; no timeout, retry, cloning or
// result disposal is added. Cancellation cannot forcibly stop uncooperative work.
// See ExecuteWithMetadata for completion/error precedence and panic semantics.
func Execute[T any](
	ctx context.Context,
	limiter *Limiter,
	operation func(context.Context) (T, error),
) (T, error) {
	return ExecuteWithMetadata(ctx, limiter, Metadata{}, operation)
}

// ExecuteWithMetadata adds copied diagnostic admission metadata to Execute.
// After admission a nil operation attempts local drop and returns ErrInvalidOutcome.
// Operation panics attempt dependency-failure completion and re-panic the original
// value. Classifier panic/invalid output attempts ignored completion and returns
// ErrClassifierPanic/ErrInvalidOutcome with the callback's value. A pre-classifier
// clock fault likewise returns ErrClock after attempted ignored completion.
//
// Otherwise the callback error wins over a Complete error; with a nil callback
// error, the completion error is returned. Partial values always remain caller-owned.
// Classifier duration uses an earlier timestamp than Complete's learning sample,
// so synchronous classifier time is included in the latter. Reap/reset races can
// make completion stale; the helper does not stop protected work or infer reentry.
func ExecuteWithMetadata[T any](
	ctx context.Context,
	limiter *Limiter,
	metadata Metadata,
	operation func(context.Context) (T, error),
) (result T, resultErr error) {
	permit, err := limiter.Acquire(ctx, metadata)
	if err != nil {
		return result, err
	}
	if operation == nil {
		_ = permit.Complete(OutcomeLocalDrop)
		return result, ErrInvalidOutcome
	}
	defer func() {
		if panicValue := recover(); panicValue != nil {
			_ = permit.Complete(OutcomeDependencyFailure)
			panic(panicValue)
		}
	}()

	result, resultErr = operation(ctx)
	finished, ok := safeNow(limiter.config.clock)
	if !ok {
		limiter.incrementClockError()
		_ = permit.Complete(OutcomeIgnored)
		return result, ErrClock
	}
	duration := max(finished.Sub(permit.started), 0)
	completion := Completion{Context: ctx, Result: result, Err: resultErr, Duration: duration}
	outcome, classifierOK := safeClassify(limiter.config.classifier, completion)
	if !classifierOK {
		limiter.incrementClassifierPanic()
		_ = permit.Complete(OutcomeIgnored)
		return result, ErrClassifierPanic
	}
	if !outcome.valid() {
		_ = permit.Complete(OutcomeIgnored)
		return result, ErrInvalidOutcome
	}
	completionErr := permit.Complete(outcome)
	if resultErr != nil {
		return result, resultErr
	}
	return result, completionErr
}

func defaultClassifier(completion Completion) Outcome {
	if completion.Context != nil && completion.Context.Err() != nil {
		return OutcomeIgnored
	}
	if completion.Err != nil {
		return OutcomeDependencyFailure
	}
	return OutcomeSuccess
}

func safeClassify(classifier Classifier, completion Completion) (outcome Outcome, ok bool) {
	defer func() { ok = recover() == nil }()
	return classifier(completion), true
}
