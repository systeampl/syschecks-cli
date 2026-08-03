package probe

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPMeasuresStatusAndTotal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	r, err := HTTP(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != 204 {
		t.Errorf("status = %d, want 204", r.Status)
	}
	if r.Total <= 0 {
		t.Errorf("total duration not measured")
	}
	// The target is an IP literal (httptest.NewServer listens on 127.0.0.1),
	// so no DNS lookup happens and DNS legitimately stays zero; Connect must
	// still be measured.
	if r.Connect <= 0 {
		t.Errorf("connect phase not measured: connect=%v", r.Connect)
	}
	if r.CertExpiry != nil {
		t.Errorf("CertExpiry = %v, want nil for plain http", r.CertExpiry)
	}
}

func TestHTTPMeasuresTLSAndCertExpiry(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	restore := setRootCAsForTest(pool)
	defer restore()

	r, err := HTTP(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != 200 {
		t.Errorf("status = %d, want 200", r.Status)
	}
	if r.TLS <= 0 {
		t.Errorf("TLS handshake duration not measured")
	}
	if r.CertExpiry == nil {
		t.Fatalf("CertExpiry not populated for https")
	}
	if r.CertExpiry.Before(time.Now()) {
		t.Errorf("cert expiry %v is in the past", r.CertExpiry)
	}
}

func TestDNSResolvesLocalhost(t *testing.T) {
	r, err := DNS("localhost")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Addrs) == 0 {
		t.Errorf("DNS(localhost) returned no addrs")
	}
	if r.Duration <= 0 {
		t.Errorf("DNS duration not measured")
	}
}

func TestTLSAgainstLocalServer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	restore := setRootCAsForTest(pool)
	defer restore()

	r, err := TLS(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if r.Version == "" {
		t.Errorf("TLS version not populated")
	}
	if r.Cipher == "" {
		t.Errorf("cipher not populated")
	}
	if r.CertExpiry.Before(time.Now()) {
		t.Errorf("cert expiry %v is in the past", r.CertExpiry)
	}
}
