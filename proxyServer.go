package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type ProxyServer struct {
	client *http.Client
	origin *url.URL
	cache  CacheClient
}

func InitProxyServer(origin *url.URL, cache CacheClient) (*ProxyServer, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Jar: jar}

	return &ProxyServer{client: client, origin: origin, cache: cache}, nil
}

func (ps *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. copy request
	// 2. change request url host to origin host
	// 3. forward request
	// 4. get reponse and forward it back to client

	if it, ok := ps.cache.Check(r.RequestURI); ok {
		ps.copyResponseHeader(w, it.responseHeader)
		w.WriteHeader(http.StatusOK)
		w.Write(it.responseBody)
		fmt.Println("X-Cache: Hit")
		return
	} else {
		fmt.Println("X-Cache: Miss")
	}

	newReq, err := ps.copyRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := ps.client.Do(newReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	{
		// check cache header
		// add to cache
		ps.cache.Add(r.RequestURI, resp.Header, rb)
	}

	// copy response
	ps.copyResponseHeader(w, resp.Header)

	// forward response
	w.WriteHeader(resp.StatusCode)
	w.Write(rb)
}

func (ps *ProxyServer) copyRequest(r *http.Request) (*http.Request, error) {
	// new request url to forward to
	reqUrl := *(ps.origin)
	reqUrl.Path = r.URL.Path
	reqUrl.RawQuery = r.URL.RawQuery

	// copy the body
	var bodyReader io.Reader
	if r.Body != nil {
		bodyReader = r.Body
	}

	// make new request
	req, err := http.NewRequest(r.Method, reqUrl.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// copy headers
	req.Header = make(http.Header)
	for key, vval := range r.Header {
		for _, v := range vval {
			req.Header.Add(key, v)
		}
	}

	return req, nil
}

func (ps *ProxyServer) copyResponseHeader(w http.ResponseWriter, respHeader http.Header) {
	// copy header
	for key, vval := range respHeader {
		for _, v := range vval {
			w.Header().Add(key, v)
		}
	}
}
