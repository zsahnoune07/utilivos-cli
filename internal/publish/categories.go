package publish

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AdminCategory is one row from GET /v1/admin/categories.
type AdminCategory struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type categoriesEnvelope struct {
	Data struct {
		Items []AdminCategory `json:"items"`
	} `json:"data"`
}

type categoriesAPIError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// FetchAdminCategories lists all categories (including archived).
func FetchAdminCategories(ctx context.Context, baseURL, apiKey string, timeout time.Duration) ([]AdminCategory, error) {
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("API base URL is empty")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("admin API key is empty")
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, base+"/v1/admin/categories", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", strings.TrimSpace(apiKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("categories request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("categories read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr categoriesAPIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Code != "" {
			return nil, fmt.Errorf("categories API %s: %s", apiErr.Error.Code, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("categories HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var env categoriesEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("categories parse JSON: %w", err)
	}
	return env.Data.Items, nil
}
