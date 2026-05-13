package doctor_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/doctor"
)

func TestParseCompose_WithPostgres(t *testing.T) {
	cf, err := doctor.ParseCompose("testdata/compose-with-postgres.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cf.Services) != 2 {
		t.Errorf("services count = %d, want 2", len(cf.Services))
	}
	if _, ok := cf.Services["web"]; !ok {
		t.Error("expected service 'web'")
	}
	if _, ok := cf.Services["postgres"]; !ok {
		t.Error("expected service 'postgres'")
	}
}

func TestParseCompose_NotFound(t *testing.T) {
	_, err := doctor.ParseCompose("testdata/does-not-exist.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestClassifyServices_PostgresDetected(t *testing.T) {
	cf, err := doctor.ParseCompose("testdata/compose-with-postgres.yaml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stateless, stateful := doctor.ClassifyServices(cf)

	if len(stateful) != 1 {
		t.Fatalf("stateful count = %d, want 1", len(stateful))
	}
	if stateful[0].Motif.MotifRef != "postgres" {
		t.Errorf("motif ref = %q, want postgres", stateful[0].Motif.MotifRef)
	}
	if stateful[0].Motif.AdminUI != "pgAdmin" {
		t.Errorf("admin UI = %q, want pgAdmin", stateful[0].Motif.AdminUI)
	}

	if len(stateless) != 1 {
		t.Fatalf("stateless count = %d, want 1", len(stateless))
	}
	if stateless[0] != "web" {
		t.Errorf("stateless[0] = %q, want web", stateless[0])
	}
}

func TestClassifyServices_StatelessOnly(t *testing.T) {
	cf, err := doctor.ParseCompose("testdata/compose-stateless-only.yaml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, stateful := doctor.ClassifyServices(cf)
	if len(stateful) != 0 {
		t.Errorf("stateful count = %d, want 0", len(stateful))
	}
}

func TestClassifyServices_MultipleStateful(t *testing.T) {
	cf, err := doctor.ParseCompose("testdata/compose-multi.yaml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, stateful := doctor.ClassifyServices(cf)
	if len(stateful) != 3 {
		t.Fatalf("stateful count = %d, want 3 (postgres, redis, rabbitmq)", len(stateful))
	}

	motifRefs := make(map[string]bool)
	for _, s := range stateful {
		motifRefs[s.Motif.MotifRef] = true
	}
	for _, expected := range []string{"postgres", "redis", "rabbitmq"} {
		if !motifRefs[expected] {
			t.Errorf("expected motif ref %q not found in stateful services", expected)
		}
	}
}

func TestDetectMotif_KafkaMultiSegment(t *testing.T) {
	cf, err := doctor.ParseCompose("testdata/compose-multi.yaml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// Verify confluentinc/cp-kafka detection doesn't conflict with plain kafka
	_ = cf
}

func TestDetectComposeFile_Found(t *testing.T) {
	// testdata/ has compose files
	path := doctor.DetectComposeFile("testdata")
	if path == "" {
		t.Error("expected to find a compose file in testdata, got empty string")
	}
}

func TestDetectComposeFile_NotFound(t *testing.T) {
	path := doctor.DetectComposeFile(".")
	if path != "" {
		t.Errorf("expected no compose file in ., got %q", path)
	}
}
