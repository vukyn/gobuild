package server

import (
	"testing"

	"github.com/vukyn/testproj/internal/config"
)

// TestProxyTrust pins the wiring that makes APP_PROXY_HEADER actually do
// something on Fiber v3.
//
// ⚠️ Fiber v3 IGNORES Config.ProxyHeader unless the connection came from a
// TRUSTED proxy: c.IP() calls IsProxyTrusted() first, which is false unless
// Config.TrustProxy is set AND a TrustProxyConfig rule (Loopback / Private /
// LinkLocal / Proxies / UnixSocket — every one of them defaults to false)
// matches the remote address. Fiber v2 needed no such wiring, so a config
// copied across from a v2 service looks right and is not.
//
// If this test is ever "simplified" away, the failure it prevents is silent:
// c.IP() returns the PROXY's own address for every caller, so every rate
// limiter sees one bucket and every audit row records one value, with no error
// logged anywhere.
func TestProxyTrust(t *testing.T) {
	cases := []struct {
		name         string
		proxyHeader  string
		wantTrust    bool
		wantLoopback bool
		wantPrivate  bool
	}{
		{
			name:        "no proxy header: trust nothing, use the socket's remote address",
			proxyHeader: "",
			wantTrust:   false,
		},
		{
			name:        "whitespace-only header is not a header",
			proxyHeader: "   ",
			wantTrust:   false,
		},
		{
			name:         "proxy header set: trust loopback and private peers",
			proxyHeader:  "Fly-Client-IP",
			wantTrust:    true,
			wantLoopback: true,
			wantPrivate:  true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := new(config.Config)
			cfg.App.ProxyHeader = testCase.proxyHeader

			trust, trustConfig := proxyTrust(cfg)

			if trust != testCase.wantTrust {
				t.Fatalf("proxyTrust(%q) TrustProxy = %v, want %v — Fiber v3 ignores ProxyHeader unless the proxy is trusted, so c.IP() would return the proxy's own address for every caller", testCase.proxyHeader, trust, testCase.wantTrust)
			}
			if trustConfig.Loopback != testCase.wantLoopback {
				t.Errorf("proxyTrust(%q) TrustProxyConfig.Loopback = %v, want %v", testCase.proxyHeader, trustConfig.Loopback, testCase.wantLoopback)
			}
			if trustConfig.Private != testCase.wantPrivate {
				t.Errorf("proxyTrust(%q) TrustProxyConfig.Private = %v, want %v", testCase.proxyHeader, trustConfig.Private, testCase.wantPrivate)
			}
			if !testCase.wantTrust && (trustConfig.LinkLocal || len(trustConfig.Proxies) > 0) {
				t.Errorf("proxyTrust(%q) returned a non-zero TrustProxyConfig while not trusting: %+v", testCase.proxyHeader, trustConfig)
			}
		})
	}
}
