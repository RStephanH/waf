package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

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

	proxy := newProxy(backendURL)

	log.Println("WAF proxy listening on :8080, forwarding to", backendURL)
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
