package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options configures multipart publish to POST /v1/admin/plugins/publish.
type Options struct {
	BaseURL   string
	APIKey    string
	Timeout   time.Duration
	Changelog string

	// Overrides; zero values fall back to manifest-derived defaults inside BuildPublishFields.
	CategorySlug     string
	CategoryName     string
	PluginStatus     string // draft | published | archived
	VersionStatus    string // draft | published | deprecated
	PricingType      string // free | pro (API); if empty, derived from manifest tier
	ShortDescription string // if empty, from manifest
	LongDescription  string // if empty, from manifest
}

// Result is the API success payload under "data".
type Result struct {
	Plugin   json.RawMessage `json:"plugin"`
	Version  json.RawMessage `json:"version"`
	Category json.RawMessage `json:"category"`
	Message  string          `json:"message"`
}

type apiEnvelope struct {
	Data Result `json:"data"`
}

type publishAPIErrorBody struct {
	Error struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Details json.RawMessage `json:"details,omitempty"`
	} `json:"error"`
}

// TierToPricing maps manifest tier to API pricing_type (free or pro only).
func TierToPricing(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "free", "":
		return "free"
	default:
		// premium, paid, pro, etc. -> pro
		return "pro"
	}
}

// Publish posts the bundle to /v1/admin/plugins/publish.
func Publish(ctx context.Context, ubbPath string, m Manifest, opts Options) (*Result, error) {
	base := strings.TrimSuffix(strings.TrimSpace(opts.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("API base URL is required (set in .env UTILIVOS_API_BASE_URL or flag --url)")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid API base URL %q", opts.BaseURL)
	}
	publishURL := base + "/v1/admin/plugins/publish"

	if err := CheckBundleSize(ubbPath); err != nil {
		return nil, err
	}

	pluginSlug := Slugify(m.Slug)
	if pluginSlug == "" {
		return nil, fmt.Errorf("manifest slug is empty after normalization")
	}
	pluginName := strings.TrimSpace(m.Name)
	if pluginName == "" {
		return nil, fmt.Errorf("manifest name is required")
	}

	catSlug := Slugify(opts.CategorySlug)
	if catSlug == "" {
		catSlug = Slugify(m.Category)
	}
	if catSlug == "" {
		catSlug = "utilities"
	}
	catName := strings.TrimSpace(opts.CategoryName)
	if catName == "" {
		catName = humanizeSlug(catSlug)
	}

	pricing := strings.TrimSpace(opts.PricingType)
	if pricing == "" {
		pricing = TierToPricing(m.Tier)
	}
	if pricing != "free" && pricing != "pro" {
		return nil, fmt.Errorf("pricing_type must be free or pro, got %q", pricing)
	}

	pluginStatus := strings.TrimSpace(opts.PluginStatus)
	if pluginStatus == "" {
		pluginStatus = "published"
	}
	versionStatus := strings.TrimSpace(opts.VersionStatus)
	if versionStatus == "" {
		versionStatus = "published"
	}

	shortDesc := strings.TrimSpace(opts.ShortDescription)
	if shortDesc == "" {
		shortDesc = strings.TrimSpace(m.Description)
	}
	longDesc := strings.TrimSpace(opts.LongDescription)
	if longDesc == "" {
		longDesc = strings.TrimSpace(m.LongDescription)
	}

	version := strings.TrimSpace(m.Version)
	if version == "" {
		return nil, fmt.Errorf("manifest version is required")
	}

	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)

	fields := map[string]string{
		"plugin_slug":    pluginSlug,
		"plugin_name":    pluginName,
		"category_slug":  catSlug,
		"category_name":  catName,
		"pricing_type":   pricing,
		"plugin_status":  pluginStatus,
		"version":        version,
		"version_status": versionStatus,
		"changelog":      strings.TrimSpace(opts.Changelog),
	}
	if shortDesc != "" {
		fields["short_description"] = shortDesc
	}
	if longDesc != "" {
		fields["long_description"] = longDesc
	}
	for k, v := range fields {
		if err := mp.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("multipart field %s: %w", k, err)
		}
	}

	f, err := os.Open(ubbPath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()

	part, err := mp.CreateFormFile("bundle", filepath.Base(ubbPath))
	if err != nil {
		return nil, fmt.Errorf("multipart file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	if err := mp.Close(); err != nil {
		return nil, err
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, publishURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mp.FormDataContentType())
	key := strings.TrimSpace(opts.APIKey)
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var apiErr publishAPIErrorBody
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Code != "" {
			return nil, fmt.Errorf("API %s: %s (%s)", apiErr.Error.Code, apiErr.Error.Message, string(apiErr.Error.Details))
		}
		return nil, fmt.Errorf("API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var env apiEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &env.Data, nil
}

func humanizeSlug(slug string) string {
	parts := strings.Split(strings.Trim(slug, "-"), "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
