.PHONY: docs fault leak stress

docs:
	go test -run='^Example' ./...
	go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./... | xargs -n 1 go doc >/dev/null

fault:
	go test -run='Test(CompletionClockFailureReleasesCapacityWithoutLearningOrGrantingInvalidPermits|ClockMetadataSamplingAndAlgorithmFaultBoundaries|ExecuteContainsClassifierPanicAndOperationPanicReleasesPermit|LimiterFaultAndEventMutationBoundaries|NewRejectsAlgorithmLifecyclePanicsAndInvalidState|QueuedAcquireContainsTimerStopPanic)$$' -count=20 .

leak:
	go test -run='^TestCanceledWaitersTerminateWithoutBackgroundWorkers$$' -count=20 .

stress:
	go test -race -run='Test(CompletionCancellationTimeoutResetSnapshotDrainRaceMatrix|FIFOAdmissionPreventsStarvationAcrossMetadataAndDurations)$$' -count=50 -timeout=5m .
