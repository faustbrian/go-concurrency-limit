.PHONY: docs stress

docs:
	./scripts/check-docs.sh
stress:
	go test -race -run='Test(CompletionCancellationTimeoutResetSnapshotDrainRaceMatrix|FIFOAdmissionPreventsStarvationAcrossMetadataAndDurations)$$' -count=50 -timeout=5m .
