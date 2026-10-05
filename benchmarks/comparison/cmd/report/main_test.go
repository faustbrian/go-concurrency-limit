package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/failsafe-go/failsafe-go/adaptivelimiter"
)

func TestCampaignsCoverRequiredWorkloadClasses(t *testing.T) {
	t.Parallel()

	workloads := campaigns()
	names := make([]string, 0, len(workloads))
	seeds := make(map[int64]struct{}, len(workloads))
	for _, candidate := range workloads {
		names = append(names, candidate.name)
		if candidate.seed == 0 {
			t.Fatalf("%s has no reproducible seed", candidate.name)
		}
		if _, exists := seeds[candidate.seed]; exists {
			t.Fatalf("seed %d is reused", candidate.seed)
		}
		seeds[candidate.seed] = struct{}{}
	}

	want := []string{
		"constant", "bursty", "ramp", "bimodal", "heavy-tail",
		"periodic", "sparse", "capacity-collapse", "class-shift",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("campaigns = %v, want %v", names, want)
	}
}

func TestComparativeCampaignsUseCommonSuccessfulCompletionSemantics(t *testing.T) {
	t.Parallel()

	for _, candidate := range campaigns() {
		random := rand.New(rand.NewSource(candidate.seed))
		for window := range candidate.windows {
			demand, capacity, rtt := candidate.model(random, window)
			if demand < 0 || capacity < 1 || rtt < 0 {
				t.Fatalf("%s window %d invalid common observation = %d/%d/%s", candidate.name, window, demand, capacity, rtt)
			}
		}
	}
}

func TestFailsafeDriverLearnsFromOneAggregateSamplePerWindow(t *testing.T) {
	t.Parallel()

	permits := make([]*recordingPermit, 8)
	values := make([]adaptivelimiter.Permit, len(permits))
	for index := range permits {
		permits[index] = &recordingPermit{}
		values[index] = permits[index]
	}
	finishFailsafeAggregate(values)
	for index, permit := range permits {
		wantRecords := 0
		wantDrops := 1
		if index == len(permits)-1 {
			wantRecords = 1
			wantDrops = 0
		}
		if permit.records != wantRecords || permit.drops != wantDrops {
			t.Fatalf("permit %d records/drops = %d/%d, want %d/%d", index, permit.records, permit.drops, wantRecords, wantDrops)
		}
	}
}

type recordingPermit struct{ records, drops int }

func (permit *recordingPermit) Record() { permit.records++ }
func (permit *recordingPermit) Drop()   { permit.drops++ }

func TestReportWritersKeepNewAndExistingFilesPrivate(t *testing.T) {
	measurements := []result{{
		workload: "constant", implementation: "local-gradient2",
		limits: []int{16, 17}, capacityTrace: []int{18, 18},
		goodput: 2, rejections: 1, capacity: 4,
		queueTotal: 6 * time.Millisecond, queueSamples: 2,
		latencies:     []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		collapseAdapt: -1, recoveryAdapt: -1,
	}}
	tests := []struct {
		name  string
		write func(*os.Root, string) error
		want  string
	}{
		{"metrics.csv", func(root *os.Root, name string) error { return writeCSV(root, name, measurements) },
			"workload,implementation,utilization,goodput,rejections,mean_queue_ms,p99_ms,collapse_adaptation_windows,recovery_adaptation_windows\nconstant,local-gradient2,0.500000,2,1,3.000,20.000,-1,-1\n"},
		{"convergence.csv", func(root *os.Root, name string) error { return writeTraceCSV(root, name, measurements) },
			"workload,window,implementation,limit,capacity\nconstant,0,local-gradient2,16,18\nconstant,1,local-gradient2,17,18\n"},
		{"convergence-constant.svg", func(root *os.Root, name string) error { return writeSVG(root, name, "constant", measurements) }, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, test.name)
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := root.Close(); err != nil {
					t.Error(err)
				}
			})
			for _, existing := range []bool{false, true} {
				if existing {
					if err := os.WriteFile(path, []byte(strings.Repeat("old synthetic report\n", 512)), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(path, 0o644); err != nil {
						t.Fatal(err)
					}
				}
				if err := test.write(root, test.name); err != nil {
					t.Fatalf("write(existing=%t): %v", existing, err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
					t.Errorf("mode(existing=%t) = %o, want 600", existing, info.Mode().Perm())
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if test.want != "" && string(data) != test.want {
					t.Errorf("content(existing=%t) = %q, want %q", existing, data, test.want)
				}
				if test.want == "" && (!strings.HasPrefix(string(data), "<svg ") ||
					!strings.HasSuffix(string(data), "</svg>") ||
					!strings.Contains(string(data), "constant convergence") ||
					!strings.Contains(string(data), "local-gradient2") ||
					strings.Contains(string(data), "old synthetic report")) {
					t.Errorf("SVG content(existing=%t) lost expected plot or retained stale data", existing)
				}
			}
		})
	}
}

func TestOutputRootCreatesPrivateDirectoryAndCloses(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "reports")
	root, err := openOutputRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil {
		_ = root.Close()
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("directory mode = %o, want 700", info.Mode().Perm())
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if file, err := openReportFile(root, "metrics.csv"); err == nil {
		_ = file.Close()
		t.Fatal("closed root accepted a report file")
	}
}

func TestReportFileInitializationMakesExistingFilePrivateAndEmpty(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "metrics.csv")
	if err := os.WriteFile(path, []byte("old synthetic report"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	file, err := openReportFile(root, "metrics.csv")
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if info.Size() != 0 || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
		t.Errorf("initialized report size/mode = %d/%o, want 0/600", info.Size(), info.Mode().Perm())
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
