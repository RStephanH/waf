package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
)

// wafHandler is the OUTER http.Handler passed to ListenAndServe.
// It replaces your ReverseProxy as the top-level handler: it inspects
// the request first, and only forwards to `next` (your existing
// ReverseProxy) if Coraza does not interrupt.
type wafHandler struct {
	waf  coraza.WAF
	next http.Handler
}

func newWAFHandler(waf coraza.WAF, next http.Handler) *wafHandler {
	return &wafHandler{waf: waf, next: next}
}

func (h *wafHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tx := h.waf.NewTransaction()
	// ProcessLogging must run even on early return (interruption or pass-through)
	// so Coraza can write audit logs for this transaction. Close() releases
	// the transaction's internal buffers.
	defer func() {
		tx.ProcessLogging()
		tx.Close()
	}()

	// --- Phase 0: connection info ---
	// RemoteAddr comes as "ip:port"; Coraza wants them split.
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr // fallback: no port info, still usable
	}
	tx.ProcessConnection(clientIP, 0, "", 0)

	// --- Phase 1: request line + headers ---
	tx.ProcessURI(r.URL.String(), r.Method, r.Proto)
	// Go's http.Request strips the "Host" header into r.Host, so it must
	// be added back manually or Coraza never sees it.
	tx.AddRequestHeader("Host", r.Host)
	for name, values := range r.Header {
		for _, v := range values {
			tx.AddRequestHeader(name, v)
		}
	}
	if it := tx.ProcessRequestHeaders(); it != nil {
		h.block(w, it)
		return
	}

	// --- Phase 2: request body ---
	// The body can only be read once. We read it fully into memory, hand
	// it to Coraza, then replace r.Body with a fresh reader so the
	// ReverseProxy behind us can still read the same content.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) > 0 {
		if _, _, err := tx.WriteRequestBody(body); err != nil {
			log.Errorf("failed writing request body to WAF: %v", err)
		}
	}
	it, err := tx.ProcessRequestBody()
	if err != nil {
		log.Errorf("coraza request body processing error: %v", err)
	}
	if it != nil {
		h.block(w, it)
		return
	}

	// Request phases are clean — forward to the backend.
	// Response-phase inspection (headers/body) is not implemented yet.
	h.next.ServeHTTP(w, r)
}

func (h *wafHandler) block(w http.ResponseWriter, it *types.Interruption) {
	log.Warnf("request blocked by WAF: rule %d, action %q", it.RuleID, it.Action)
	http.Error(w, "request blocked", http.StatusForbidden)
}

func newProxy(backendURL *url.URL) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(backendURL)
			pr.SetXForwarded() // safely sets X-Forwarded-For/Host/Proto, overwriting any client-supplied value
			log.Printf("[REQUEST] %s %s  -> %s \n", pr.In.Method, pr.In.URL.Path, backendURL.Host)
		},
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		log.Printf("[RESPONSE] %d for %s", resp.StatusCode, resp.Request.URL)
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[ERROR] proxying %s %s: %v", r.Method, r.URL.Path, err)
		w.WriteHeader(http.StatusBadGateway)
	}

	return proxy
}

func main() {
	backendURL, err := url.Parse("http://127.0.0.1:3000") // Juice Shop address
	if err != nil {
		log.Fatalf("invalid backend URL: %v", err)
	}

	waf, err := coraza.NewWAF(
		coraza.NewWAFConfig().
			WithDirectivesFromFile("coraza.conf-recommended").
			WithDirectivesFromFile("coreruleset/crs-setup.conf.example").
			WithDirectivesFromFile("coreruleset/rules/*.conf").
			WithDirectives("SecRuleEngine On"),
	)
	if err != nil {
		log.Fatalf("failed to initialize Coraza WAF: %v", err)
	}

	proxy := newProxy(backendURL)
	handler := newWAFHandler(waf, proxy)

	log.Infof("WAF proxy listening on :8080, forwarding to %s", backendURL)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
