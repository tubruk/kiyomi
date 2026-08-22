package dnsresolver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseList tests ---

func TestParseList_Empty(t *testing.T) {
	specs, legacy, err := ParseList("")
	require.NoError(t, err)
	assert.Nil(t, specs)
	assert.Nil(t, legacy)
}

func TestParseList_DNSURL(t *testing.T) {
	specs, legacy, err := ParseList("dns://1.1.1.1")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Len(t, legacy, 0)
	assert.Equal(t, "dns", specs[0].Scheme)
	assert.Equal(t, "1.1.1.1", specs[0].Host)
	assert.Equal(t, 53, specs[0].Port)
}

func TestParseList_DNSURLWithPort(t *testing.T) {
	specs, _, err := ParseList("dns://1.1.1.1:5353")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, 5353, specs[0].Port)
}

func TestParseList_DoT(t *testing.T) {
	specs, _, err := ParseList("tls://1.1.1.1")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "tls", specs[0].Scheme)
	assert.Equal(t, "1.1.1.1", specs[0].Host)
	assert.Equal(t, 853, specs[0].Port)
}

func TestParseList_DoTWithPort(t *testing.T) {
	specs, _, err := ParseList("tls://1.1.1.1:8530")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, 8530, specs[0].Port)
}

func TestParseList_DoH(t *testing.T) {
	specs, _, err := ParseList("https://dns.google/dns-query")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "https", specs[0].Scheme)
	assert.Equal(t, "dns.google", specs[0].Host)
	assert.Equal(t, 443, specs[0].Port)
	assert.Equal(t, "/dns-query", specs[0].Path)
}

func TestParseList_DoHWithCustomPath(t *testing.T) {
	specs, _, err := ParseList("https://dns.nextdns.io/my-profile")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "/my-profile", specs[0].Path)
}

func TestParseList_DoHDefaultPath(t *testing.T) {
	specs, _, err := ParseList("https://dns.google")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "/dns-query", specs[0].Path)
}

func TestParseList_LegacyHostPort(t *testing.T) {
	specs, legacy, err := ParseList("1.1.1.1:53")
	require.NoError(t, err)
	require.Len(t, specs, 0)
	require.Len(t, legacy, 1)
	assert.Equal(t, "1.1.1.1:53", legacy[0])
}

func TestParseList_LegacyHostPortNoScheme(t *testing.T) {
	specs, legacy, err := ParseList("custom.resolver.local:5353")
	require.NoError(t, err)
	require.Len(t, specs, 0)
	require.Len(t, legacy, 1)
	assert.Equal(t, "custom.resolver.local:5353", legacy[0])
}

func TestParseList_Bootstrap(t *testing.T) {
	specs, _, err := ParseList("tls://resolver.example.com?bootstrap=192.168.1.1")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, []string{"192.168.1.1"}, specs[0].Bootstrap)
}

func TestParseList_BootstrapMultiple(t *testing.T) {
	specs, _, err := ParseList("https://resolver.example.com/dns-query?bootstrap=192.168.1.1&bootstrap=192.168.1.2")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, []string{"192.168.1.1", "192.168.1.2"}, specs[0].Bootstrap)
}

func TestParseList_Malformed(t *testing.T) {
	_, _, err := ParseList("dns://[invalid]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dnsresolver: parse")
}

func TestParseList_Multiple(t *testing.T) {
	raw := "dns://1.1.1.1, tls://1.0.0.1, https://dns.google/dns-query, 8.8.8.8:53"
	specs, legacy, err := ParseList(raw)
	require.NoError(t, err)
	require.Len(t, specs, 3)
	require.Len(t, legacy, 1)
	assert.Equal(t, "dns", specs[0].Scheme)
	assert.Equal(t, "tls", specs[1].Scheme)
	assert.Equal(t, "https", specs[2].Scheme)
	assert.Equal(t, "8.8.8.8:53", legacy[0])
}

func TestParseList_Whitespace(t *testing.T) {
	specs, legacy, err := ParseList("  dns://1.1.1.1  ,  1.1.1.2:53  ")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Len(t, legacy, 1)
	assert.Equal(t, "1.1.1.1", specs[0].Host)
	assert.Equal(t, "1.1.1.2:53", legacy[0])
}

// --- LoadFromEnv tests ---

func TestLoadFromEnv_Unset(t *testing.T) {
	orig := os.Getenv("KIYOMI_TEST_DNS_RESOLVERS")
	os.Unsetenv("KIYOMI_TEST_DNS_RESOLVERS")
	defer func() {
		if orig != "" {
			os.Setenv("KIYOMI_TEST_DNS_RESOLVERS", orig)
		}
	}()

	result := LoadFromEnv("KIYOMI_TEST_DNS_RESOLVERS")
	assert.Nil(t, result)
}

func TestLoadFromEnv_Prefixed(t *testing.T) {
	os.Setenv("KIYOMI_TEST_DNS_RESOLVERS", "dns://1.1.1.1,1.1.1.2:53")
	defer os.Unsetenv("KIYOMI_TEST_DNS_RESOLVERS")

	result := LoadFromEnv("KIYOMI_TEST_DNS_RESOLVERS")
	require.Len(t, result, 2)
	assert.Equal(t, "dns://1.1.1.1:53", result[0])
	assert.Equal(t, "1.1.1.2:53", result[1])
}

func TestLoadFromEnv_Invalid(t *testing.T) {
	os.Setenv("KIYOMI_TEST_DNS_RESOLVERS", "dns://[invalid]")
	defer os.Unsetenv("KIYOMI_TEST_DNS_RESOLVERS")

	result := LoadFromEnv("KIYOMI_TEST_DNS_RESOLVERS")
	assert.Nil(t, result)
}

// --- Integration tests using test servers ---

func TestIntegration_PlainDNS(t *testing.T) {
	// Register handler globally.
	dns.HandleFunc("example.com.", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.RecursionDesired = true
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("127.0.0.1"),
		})
		w.WriteMsg(m)
	})
	defer dns.HandleRemove("example.com.")

	// Start a real TCP listener to be the target of the dial.
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer targetLn.Close()
	targetPort := targetLn.Addr().(*net.TCPAddr).Port

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	host := pc.LocalAddr().(*net.UDPAddr).IP.String()
	port := pc.LocalAddr().(*net.UDPAddr).Port

	server := &dns.Server{PacketConn: pc}
	go server.ActivateAndServe()
	defer server.Shutdown()

	time.Sleep(50 * time.Millisecond)

	dialFn := DialFunc([]Spec{{Scheme: "dns", Host: host, Port: port}})
	ctx := context.Background()
	conn, err := dialFn(ctx, "tcp", fmt.Sprintf("example.com.:%d", targetPort))
	require.NoError(t, err)
	defer conn.Close()
}

func TestIntegration_DoT(t *testing.T) {
	// Start a real TCP listener to be the target of the dial.
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer targetLn.Close()
	targetPort := targetLn.Addr().(*net.TCPAddr).Port

	// Generate a self-signed certificate.
	cert, key, err := generateSelfSignedCert("localhost")
	require.NoError(t, err)
	tlsCert, err := tls.X509KeyPair(cert, key)
	require.NoError(t, err)

	// Register DNS handler globally.
	dns.HandleFunc("example.com.", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.RecursionDesired = true
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("127.0.0.1"),
		})
		w.WriteMsg(m)
	})
	defer dns.HandleRemove("example.com.")

	// Start UDP DNS server (miekg/dns uses UDP for the backend).
	udpL, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	udpServerAddr := udpL.LocalAddr().(*net.UDPAddr)
	udpServer := &dns.Server{Addr: udpServerAddr.String(), Net: "udp"}
	go udpServer.ActivateAndServe()
	defer udpServer.Shutdown()
	time.Sleep(50 * time.Millisecond)

	// Start TLS listener for DoT.
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	require.NoError(t, err)
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleDotConn(conn)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Now test lookup with bootstrap.
	tcpAddr := ln.Addr().(*net.TCPAddr)
	host := tcpAddr.IP.String()
	port := tcpAddr.Port

	dialFn := DialFunc([]Spec{{Scheme: "tls", Host: host, Port: port, Bootstrap: []string{host}}})
	ctx := context.Background()
	conn, err := dialFn(ctx, "tcp", fmt.Sprintf("example.com.:%d", targetPort))
	require.NoError(t, err)
	conn.Close()
}

func handleDotConn(c net.Conn) {
	defer c.Close()
	// Read length prefix.
	lenBuf := make([]byte, 2)
	_, err := io.ReadFull(c, lenBuf)
	if err != nil {
		return
	}
	msgLen := int(lenBuf[0])<<8 | int(lenBuf[1])
	buf := make([]byte, msgLen)
	_, err = io.ReadFull(c, buf)
	if err != nil {
		return
	}
	req := new(dns.Msg)
	err = req.Unpack(buf)
	if err != nil {
		return
	}
	m := new(dns.Msg)
	m.SetReply(req)
	m.RecursionDesired = true
	m.Answer = append(m.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
		A:   net.ParseIP("127.0.0.1"),
	})
	respBuf, _ := m.Pack()
	respLen := make([]byte, 2)
	respLen[0] = byte(len(respBuf) >> 8)
	respLen[1] = byte(len(respBuf))
	c.Write(append(respLen, respBuf...))
}

func TestIntegration_DoH(t *testing.T) {
	// Start a real TCP listener to be the target of the dial.
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer targetLn.Close()
	targetPort := targetLn.Addr().(*net.TCPAddr).Port

	// Start an HTTP server that handles DoH requests.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var wire []byte
		var err error

		if r.Method == "GET" {
			dnsParam := r.URL.Query().Get("dns")
			wire, err = base64.RawURLEncoding.DecodeString(dnsParam)
		} else if r.Method == "POST" {
			wire, err = io.ReadAll(r.Body)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		req := new(dns.Msg)
		if err := req.Unpack(wire); err != nil {
			http.Error(w, "bad dns msg", http.StatusBadRequest)
			return
		}

		m := new(dns.Msg)
		m.SetReply(req)
		m.RecursionDesired = true
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("127.0.0.1"),
		})

		respWire, _ := m.Pack()
		w.Header().Set("Content-Type", "application/dns-message")
		w.Write(respWire)
	})

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	// Extract host and port from the test server's listener address.
	serverAddr := server.Listener.Addr().(*net.TCPAddr)
	host := serverAddr.IP.String()
	port := serverAddr.Port

	// Test DoH lookup using HTTP (no TLS) with explicit port from test server.
	dialFn := DialFunc([]Spec{{Scheme: "https", Host: host, Port: port, Path: "/", Bootstrap: []string{host}}})
	ctx := context.Background()
	conn, err := dialFn(ctx, "tcp", fmt.Sprintf("example.com.:%d", targetPort))
	require.NoError(t, err)
	conn.Close()
}

func TestIntegration_Bootstrap(t *testing.T) {
	// Test that parsing works correctly for bootstrap scenario.
	specs, _, err := ParseList("tls://resolver.example.com?bootstrap=127.0.0.1")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "resolver.example.com", specs[0].Host)
	assert.Equal(t, []string{"127.0.0.1"}, specs[0].Bootstrap)
}

// --- DialFuncFromURLs tests ---

func TestDialFuncFromURLs(t *testing.T) {
	dialFn, err := DialFuncFromURLs([]string{"dns://1.1.1.1"})
	require.NoError(t, err)
	require.NotNil(t, dialFn)
}

func TestDialFuncFromURLs_Invalid(t *testing.T) {
	_, err := DialFuncFromURLs([]string{"http://192.168.0.%31/"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dnsresolver: parse URL")
}

// --- NewResolver tests ---

func TestNewResolver(t *testing.T) {
	specs := []Spec{{Scheme: "dns", Host: "1.1.1.1", Port: 53}}
	resolver := NewResolver(specs)
	require.NotNil(t, resolver)
	assert.True(t, resolver.PreferGo)
}

func TestDoHResolver_ConnectionReuse(t *testing.T) {
	requestCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var wire []byte
		var err error

		if r.Method == "GET" {
			dnsParam := r.URL.Query().Get("dns")
			wire, err = base64.RawURLEncoding.DecodeString(dnsParam)
		} else if r.Method == "POST" {
			wire, err = io.ReadAll(r.Body)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		req := new(dns.Msg)
		if err := req.Unpack(wire); err != nil {
			http.Error(w, "bad dns msg", http.StatusBadRequest)
			return
		}

		m := new(dns.Msg)
		m.SetReply(req)
		m.RecursionDesired = true
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("127.0.0.1"),
		})

		respWire, _ := m.Pack()
		w.Header().Set("Content-Type", "application/dns-message")
		w.Write(respWire)
	})

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	serverAddr := server.Listener.Addr().(*net.TCPAddr)
	host := serverAddr.IP.String()
	port := serverAddr.Port

	r := newDoHResolver(host, port, "/", []string{host})
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		ip, err := r.lookup(ctx, "example.com.")
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1", ip.String())
	}
	assert.Equal(t, 5, requestCount)
}

// --- helper ---

// generateSelfSignedCert generates a self-signed certificate for testing.
func generateSelfSignedCert(host string) ([]byte, []byte, error) {
	// Generate RSA key for the certificate (Go 1.22+ compatible).
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)})

	return certPEM, keyPEM, nil
}
