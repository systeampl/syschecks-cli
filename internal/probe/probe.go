// Package probe implements client-side HTTP/DNS/TLS diagnostics for the
// syschecks CLI: a single GET timed via httptrace.ClientTrace, a plain DNS
// lookup, and a raw TLS handshake. It deliberately talks to the target
// directly (net/http, crypto/tls, net) rather than through the SDK — the
// whole point is to measure what a real client sees.
package probe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
)

// HTTPResult is the timing breakdown and outcome of a single measured GET.
type HTTPResult struct {
	Status     int
	DNS        time.Duration
	Connect    time.Duration
	TLS        time.Duration
	TTFB       time.Duration
	Total      time.Duration
	CertExpiry *time.Time
}

// rootCAsOverride lets tests point the TLS verification done by HTTP and TLS
// at a private CA pool (e.g. an httptest.NewTLSServer's self-signed cert)
// instead of the system trust store. Production code never sets it, so
// verification always uses the system trust store (nil RootCAs) in normal
// operation. Not safe for concurrent test use, but package tests run
// sequentially.
var rootCAsOverride *x509.CertPool

// setRootCAsForTest overrides rootCAsOverride for the duration of a test and
// returns a func that restores the previous value; callers should defer it.
func setRootCAsForTest(pool *x509.CertPool) (restore func()) {
	prev := rootCAsOverride
	rootCAsOverride = pool
	return func() { rootCAsOverride = prev }
}

// HTTP performs a single GET against url, measuring DNS lookup, TCP connect,
// TLS handshake, time-to-first-byte, and total wall time via
// httptrace.ClientTrace. For https targets, CertExpiry is the leaf
// certificate's NotAfter; it stays nil for plain http.
func HTTP(ctx context.Context, url string) (HTTPResult, error) {
	var res HTTPResult
	var start, dnsStart, connectStart, tlsStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:  func(httptrace.DNSDoneInfo) { res.DNS = time.Since(dnsStart) },
		ConnectStart: func(string, string) {
			connectStart = time.Now()
		},
		ConnectDone: func(string, string, error) {
			res.Connect = time.Since(connectStart)
		},
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			res.TLS = time.Since(tlsStart)
			if err == nil && len(cs.PeerCertificates) > 0 {
				expiry := cs.PeerCertificates[0].NotAfter
				res.CertExpiry = &expiry
			}
		},
		GotFirstResponseByte: func() { res.TTFB = time.Since(start) },
	}
	traceCtx := httptrace.WithClientTrace(ctx, trace)

	req, err := http.NewRequestWithContext(traceCtx, http.MethodGet, url, nil)
	if err != nil {
		return HTTPResult{}, fmt.Errorf("building request: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: rootCAsOverride},
		},
	}

	start = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return HTTPResult{}, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return HTTPResult{}, fmt.Errorf("reading response body: %w", err)
	}
	res.Total = time.Since(start)
	res.Status = resp.StatusCode
	return res, nil
}

// DNSResult is the outcome of a plain hostname lookup.
type DNSResult struct {
	Addrs    []string
	Duration time.Duration
}

// DNS resolves host via net.DefaultResolver.LookupHost, timing the lookup.
// ctx bounds the lookup: an unreachable resolver would otherwise stall for as
// long as the OS resolver decides to.
func DNS(ctx context.Context, host string) (DNSResult, error) {
	start := time.Now()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	dur := time.Since(start)
	if err != nil {
		return DNSResult{}, fmt.Errorf("resolving %s: %w", host, err)
	}
	return DNSResult{Addrs: addrs, Duration: dur}, nil
}

// TLSResult is the negotiated connection parameters and leaf certificate
// info from a raw TLS handshake.
type TLSResult struct {
	Version    string
	Cipher     string
	CertExpiry time.Time
	DNSNames   []string
}

// TLS dials hostPort ("host:port") with tls.Dial and reports the negotiated
// protocol version, cipher suite, and the leaf certificate's expiry and SAN
// DNS names.
// ctx bounds the dial and handshake: a peer that accepts the connection and
// never completes the handshake would otherwise hold the command open.
func TLS(ctx context.Context, hostPort string) (TLSResult, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return TLSResult{}, fmt.Errorf("parsing host:port %q: %w", hostPort, err)
	}

	d := &tls.Dialer{Config: &tls.Config{ServerName: host, RootCAs: rootCAsOverride}}
	nconn, err := d.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return TLSResult{}, fmt.Errorf("dialing %s: %w", hostPort, err)
	}
	conn := nconn.(*tls.Conn)
	defer conn.Close()

	cs := conn.ConnectionState()
	res := TLSResult{
		Version: tlsVersionName(cs.Version),
		Cipher:  tls.CipherSuiteName(cs.CipherSuite),
	}
	if len(cs.PeerCertificates) > 0 {
		cert := cs.PeerCertificates[0]
		res.CertExpiry = cert.NotAfter
		res.DNSNames = cert.DNSNames
	}
	return res, nil
}

// tlsVersionName maps a tls.VersionTLS* constant to its human-readable name.
func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}
