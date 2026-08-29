.PHONY: docs stress

docs:
	go test -run='^Example' ./...
	go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./... | xargs -n 1 go doc >/dev/null

stress:
	go test -race -run='Test(CompletionCancellationTimeoutResetSnapshotDrainRaceMatrix|FIFOAdmissionPreventsStarvationAcrossMetadataAndDurations)$$' -count=50 -timeout=5m .
