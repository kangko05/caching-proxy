package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

func main() {
	port, origin, err := parseFlags()
	if err != nil {
		fmt.Println(err)
		return
	}

	cacheClient := InitCacheClient(FIFO, 100)

	go cacheClient.Run()
	defer cacheClient.Stop()

	proxyServer, err := InitProxyServer(origin, cacheClient)
	if err != nil {
		fmt.Printf("failed to init proxy server: %v\n", err)
		return
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: proxyServer,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Println("failed to start the server")
			return
		}
	}()

	// for gracefull shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("server shutdown err: %v", err)
		return
	}

	fmt.Println("server down")
}

func parseFlags() (int, *url.URL, error) {
	port := flag.Int("port", 3000, "proxy server listens to <PORT> (default: 3000)")
	origin := flag.String("origin", "", "proxy forwards request to <ORIGIN>")

	flag.Parse()

	if !hasUrlScheme(*origin) {
		return -1, nil, fmt.Errorf("url needs to contain its scheme (http or https)\ninvalid origin url: %s", *origin)
	}

	originParsed, err := url.Parse(*origin)
	if err != nil {
		return -1, nil, fmt.Errorf("failed to parse origin url: %v", err)
	}

	return *port, originParsed, nil
}

func hasUrlScheme(urlStr string) bool {
	re := regexp.MustCompile(`^(http|https)://`)
	return re.MatchString(urlStr)
}
