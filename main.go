package main

import (
	"io"
	"log"
	"net"
	"net/http"
)

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	req.Header = r.Header
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, "Unable to connect to host", http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()

	w.WriteHeader(http.StatusOK)
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, "Hijacking failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	go io.Copy(destConn, clientConn)
	io.Copy(clientConn, destConn)
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodConnect {
        handleTunneling(w, r)
        return
    }
    handleHTTP(w, r)
}

func main() {
    http.HandleFunc("/", handleProxy)

    log.Println("Proxy running on :8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}
