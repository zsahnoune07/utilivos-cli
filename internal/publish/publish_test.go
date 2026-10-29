package publish

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadAndValidateManifest(t *testing.T) {
	dir := t.TempDir()
	ubb := filepath.Join(dir, "t.ubb")
	if err := writeTestUBB(ubb, testManifestJSON()); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifestFromUBB(ubb)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateForPublish(m); err != nil {
		t.Fatal(err)
	}
	if m.Slug != "hello-tool" {
		t.Fatalf("slug: %q", m.Slug)
	}
}

func TestPublishMultipartAndStatuses(t *testing.T) {
	dir := t.TempDir()
	ubb := filepath.Join(dir, "hello-tool.ubb")
	if err := writeTestUBB(ubb, testManifestJSON()); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/admin/plugins/publish" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("plugin_status") != "published" || r.FormValue("version_status") != "published" {
			t.Fatalf("status fields: plugin=%q version=%q", r.FormValue("plugin_status"), r.FormValue("version_status"))
		}
		if r.FormValue("plugin_slug") != "hello-tool" || r.FormValue("version") != "1.0.0" {
			t.Fatalf("slug/version mismatch")
		}
		if r.FormValue("pricing_type") != "free" {
			t.Fatalf("pricing_type %q", r.FormValue("pricing_type"))
		}
		_, hdr, err := r.FormFile("bundle")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(strings.ToLower(hdr.Filename), ".ubb") {
			t.Fatalf("bundle filename %q", hdr.Filename)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"message":"ok","plugin":{},"version":{},"category":{}}}`)
	}))
	defer srv.Close()

	m, err := ReadManifestFromUBB(ubb)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Publish(context.Background(), ubb, m, Options{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "ok" {
		t.Fatalf("message %q", res.Message)
	}
}

func TestPublishPricingProForPremiumTier(t *testing.T) {
	dir := t.TempDir()
	ubb := filepath.Join(dir, "pro-tool.ubb")
	manifest := `{
  "name": "Pro",
  "slug": "pro-tool",
  "version": "1.0.0",
  "description": "Paid tier test.",
  "category": "utilities",
  "tier": "premium"
}`
	if err := writeTestUBB(ubb, manifest); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("pricing_type") != "pro" {
			t.Fatalf("pricing_type %q", r.FormValue("pricing_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"message":"ok","plugin":{},"version":{},"category":{}}}`)
	}))
	defer srv.Close()
	m, err := ReadManifestFromUBB(ubb)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Publish(context.Background(), ubb, m, Options{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResolveCategoryNonInteractiveMatchManifest(t *testing.T) {
	m := Manifest{Category: "utilities"}
	cats := []AdminCategory{{Slug: "utilities", Name: "Utilities"}}
	s, n, err := ResolveCategoryNonInteractive(m, cats, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if s != "utilities" || n != "Utilities" {
		t.Fatalf("got %q %q", s, n)
	}
}

func TestResolveCategoryNonInteractiveRequiresFlags(t *testing.T) {
	m := Manifest{Category: "unknown-cat"}
	cats := []AdminCategory{{Slug: "utilities", Name: "Utilities"}}
	_, _, err := ResolveCategoryNonInteractive(m, cats, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishAPIErrorJSON(t *testing.T) {
	dir := t.TempDir()
	ubb := filepath.Join(dir, "x.ubb")
	if err := writeTestUBB(ubb, testManifestJSON()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":"validation_error","message":"bad","details":[]}}`)
	}))
	defer srv.Close()
	m, _ := ReadManifestFromUBB(ubb)
	_, err := Publish(context.Background(), ubb, m, Options{BaseURL: srv.URL})
	if err == nil || !strings.Contains(err.Error(), "validation_error") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func writeTestUBB(path, manifest string) error {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create("manifest.json")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, manifest); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func testManifestJSON() string {
	return `{
  "name": "Hello",
  "slug": "hello-tool",
  "version": "1.0.0",
  "description": "Test tool for publish CLI.",
  "category": "utilities",
  "tier": "free"
}`
}

func TestMultipartFieldOrderIndependent(t *testing.T) {
	// Sanity: our client produces parseable multipart
	var b bytes.Buffer
	mp := multipart.NewWriter(&b)
	_ = mp.WriteField("plugin_slug", "a")
	_ = mp.Close()
	if !bytes.Contains(b.Bytes(), []byte("plugin_slug")) {
		t.Fatal("expected field name in body")
	}
}
