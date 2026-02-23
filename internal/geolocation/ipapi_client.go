package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"ipscope/internal/model"
)

const defaultBaseURL = "http://ip-api.com/json"

const defaultMinRequestInterval = 1500 * time.Millisecond

type Resolver interface {
	ResolveDatacenter(ctx context.Context, endpoint string) (model.DatacenterInfo, error)
}

type APIClient struct {
	httpClient *http.Client
	baseURL    string

	mu           sync.Mutex
	minInterval  time.Duration
	blockedUntil time.Time
	lastRequest  time.Time
}

type ipAPIResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message,omitempty"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	Query       string  `json:"query"`
}

func NewAPIClient(timeout time.Duration) *APIClient {
	return &APIClient{
		httpClient:  &http.Client{Timeout: timeout},
		baseURL:     defaultBaseURL,
		minInterval: defaultMinRequestInterval,
	}
}

func (c *APIClient) ResolveDatacenter(ctx context.Context, endpoint string) (model.DatacenterInfo, error) {
	if err := c.waitForAllowance(ctx); err != nil {
		return model.DatacenterInfo{}, err
	}

	requestURL := fmt.Sprintf("%s/%s", c.baseURL, url.PathEscape(strings.TrimSpace(endpoint)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return model.DatacenterInfo{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.DatacenterInfo{}, fmt.Errorf("request geolocation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			xRL := resp.Header.Get("X-Rl")
			xTTL := resp.Header.Get("X-Ttl")
			c.applyRateLimitWindow(xTTL)
			return model.DatacenterInfo{}, fmt.Errorf("geolocation API returned status %d (rate limited, X-Rl=%q, X-Ttl=%q)", resp.StatusCode, xRL, xTTL)
		}

		return model.DatacenterInfo{}, fmt.Errorf("geolocation API returned status %d", resp.StatusCode)
	}

	c.applyRemainingWindow(resp.Header.Get("X-Rl"), resp.Header.Get("X-Ttl"))

	var parsed ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return model.DatacenterInfo{}, fmt.Errorf("decode geolocation API response: %w", err)
	}

	if parsed.Status != "success" {
		reason := strings.TrimSpace(parsed.Message)
		if reason == "" {
			reason = "unknown"
		}

		if strings.Contains(strings.ToLower(reason), "too many requests") {
			xRL := resp.Header.Get("X-Rl")
			xTTL := resp.Header.Get("X-Ttl")
			c.applyRateLimitWindow(xTTL)
			return model.DatacenterInfo{}, fmt.Errorf("geolocation API error: %s (rate limited, X-Rl=%q, X-Ttl=%q)", reason, xRL, xTTL)
		}

		return model.DatacenterInfo{}, fmt.Errorf("geolocation API error: %s", reason)
	}

	datacenter := firstNonEmpty(parsed.Org, parsed.ISP, fmt.Sprintf("%s-%s", parsed.City, parsed.RegionName), parsed.City, "unknown")

	return model.DatacenterInfo{
		Datacenter: datacenter,
		City:       parsed.City,
		Region:     parsed.RegionName,
		Country:    parsed.Country,
		Latitude:   parsed.Latitude,
		Longitude:  parsed.Longitude,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func (c *APIClient) waitForAllowance(ctx context.Context) error {
	for {
		wait := c.nextWaitDuration()
		if wait <= 0 {
			return nil
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("wait for geolocation rate limit allowance: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (c *APIClient) nextWaitDuration() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if c.blockedUntil.After(now) {
		return c.blockedUntil.Sub(now)
	}

	if c.lastRequest.IsZero() {
		c.lastRequest = now
		return 0
	}

	nextAllowed := c.lastRequest.Add(c.minInterval)
	if nextAllowed.After(now) {
		return nextAllowed.Sub(now)
	}

	c.lastRequest = now
	return 0
}

func (c *APIClient) applyRemainingWindow(xRL string, xTTL string) {
	if strings.TrimSpace(xRL) != "0" {
		return
	}

	c.applyRateLimitWindow(xTTL)
}

func (c *APIClient) applyRateLimitWindow(xTTL string) {
	ttl, err := strconv.Atoi(strings.TrimSpace(xTTL))
	if err != nil || ttl <= 0 {
		return
	}

	until := time.Now().Add(time.Duration(ttl) * time.Second)

	c.mu.Lock()
	if until.After(c.blockedUntil) {
		c.blockedUntil = until
	}
	c.mu.Unlock()
}
