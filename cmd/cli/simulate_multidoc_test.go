//go:build !runtime && !gateway

package cli

import "testing"

const twoWebsitesOneCert = `
apiVersion: example.com/v1
kind: Website
metadata:
  name: site-b
  namespace: default
spec:
  domain: shared.example.com
---
apiVersion: example.com/v1
kind: Website
metadata:
  name: site-a
  namespace: default
spec:
  domain: shared.example.com
---
apiVersion: example.com/v1
kind: Certificate
metadata:
  name: cert-a
  namespace: default
spec:
  domain: a.example.com
`

func TestParseMultiDocCRs_GroupsBySameKind(t *testing.T) {
	crs := parseMultiDocCRs([]byte(twoWebsitesOneCert))

	if len(crs["website"]) != 2 {
		t.Fatalf("expected 2 Website docs, got %d", len(crs["website"]))
	}
	if len(crs["certificate"]) != 1 {
		t.Fatalf("expected 1 Certificate doc, got %d", len(crs["certificate"]))
	}
	// File order preserved — site-b first, site-a second.
	if got := crs["website"][0].GetName(); got != "site-b" {
		t.Errorf("expected first Website doc to be site-b, got %s", got)
	}
	if got := crs["website"][1].GetName(); got != "site-a" {
		t.Errorf("expected second Website doc to be site-a, got %s", got)
	}
}

func TestResolveCRInputs_FirstDocIsReconciled(t *testing.T) {
	crs := parseMultiDocCRs([]byte(twoWebsitesOneCert))

	in, ok := resolveCRInputs(crs, "Website")
	if !ok {
		t.Fatal("expected Website inputs to resolve")
	}
	if in.cr.GetName() != "site-b" {
		t.Errorf("expected cr under test to be the first doc (site-b), got %s", in.cr.GetName())
	}
	if len(in.existing) != 1 || in.existing[0].GetName() != "site-a" {
		t.Fatalf("expected exactly one existing instance (site-a), got %+v", in.existing)
	}
	if len(in.peers) != 1 || in.peers["certificate"] == nil {
		t.Fatalf("expected certificate as the only peer, got %+v", in.peers)
	}
	if _, hasSelfPeer := in.peers["website"]; hasSelfPeer {
		t.Error("website should not appear in its own peers map")
	}
}

func TestResolveCRInputs_KindNotPresent(t *testing.T) {
	crs := parseMultiDocCRs([]byte(twoWebsitesOneCert))

	if _, ok := resolveCRInputs(crs, "Deployment"); ok {
		t.Error("expected resolveCRInputs to report not-found for a kind absent from the file")
	}
}

func TestResolveCRInputs_SingleDoc_NoExistingInstances(t *testing.T) {
	crs := parseMultiDocCRs([]byte(twoWebsitesOneCert))

	in, ok := resolveCRInputs(crs, "Certificate")
	if !ok {
		t.Fatal("expected Certificate inputs to resolve")
	}
	if len(in.existing) != 0 {
		t.Errorf("single-doc kind should have no existing instances, got %+v", in.existing)
	}
	if _, hasSelfPeer := in.peers["certificate"]; hasSelfPeer {
		t.Error("certificate should not appear in its own peers map")
	}
	if in.peers["website"] == nil {
		t.Error("expected website as a peer (first Website doc)")
	}
}
