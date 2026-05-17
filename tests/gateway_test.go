package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/transport"
)

func TestGatewayHealthAndPing(t *testing.T) {
	baseURL, stop := startGatewayForTest(t)
	defer stop()

	assertStatus(t, baseURL+"/health", "", http.StatusOK)
	assertStatus(t, baseURL+"/api/v1/ping", "", http.StatusOK)
}

func TestGatewayInvalidJWT(t *testing.T) {
	baseURL, stop := startGatewayForTest(t)
	defer stop()

	assertStatus(t, baseURL+"/api/v1/profile", "Bearer invalid", http.StatusUnauthorized)
}

func TestGatewayDocs(t *testing.T) {
	baseURL, stop := startGatewayForTest(t)
	defer stop()

	assertStatus(t, baseURL+"/docs", "", http.StatusOK)

	resp, err := http.Get(baseURL + "/openapi.json")
	if err != nil {
		t.Fatalf("GET /openapi.json: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("parse openapi json: %v", err)
	}

	if payload["openapi"] == "" {
		t.Fatalf("openapi field missing")
	}
}

func startGatewayForTest(t *testing.T) (string, func()) {
	t.Helper()

	addr := freeAddr(t)
	cfg := config.Config{
		HTTPAddr:              addr,
		GatewayJWTToken:       "dev-gateway-token",
		GatewayTimeoutSeconds: 5,
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := transport.NewServer(cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = server.Run(ctx)
	}()

	waitForServer(t, "http://"+addr+"/health")

	return "http://" + addr, func() {
		cancel()
		time.Sleep(100 * time.Millisecond)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on random port: %v", err)
	}
	defer ln.Close()

	return ln.Addr().String()
}

func waitForServer(t *testing.T, healthURL string) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("server did not become healthy in time")
}

func assertStatus(t *testing.T, url, authHeader string, want int) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != want {
		t.Fatalf("expected status %d for %s, got %d", want, url, resp.StatusCode)
	}
}
