package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

type ExternalIPChecker struct {
	client     *http.Client
	ExternalIP string
}

func NewExternalIPChecker() *ExternalIPChecker {
	return &ExternalIPChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (e *ExternalIPChecker) HasNewExternalIP() bool {
	var res *http.Response
	var body []byte
	var err error

	if res, err = e.client.Get("https://1.1.1.1/cdn-cgi/trace"); err != nil {
		return false
	}

	if body, err = ioutil.ReadAll(res.Body); err != nil {
		return false
	}

	for _, line := range strings.Split(string(body), "\n") {
		kv := strings.Split(line, "=")
		if kv[0] == "ip" && e.ExternalIP != kv[1] {
			e.ExternalIP = kv[1]
			return true
		}
	}

	return false
}

func main() {
	externalIPChecker := NewExternalIPChecker()

	var api *cloudflare.API
	var err error

	if api, err = cloudflare.NewWithAPIToken(os.Getenv("CF_API_KEY")); err != nil {
		log.Printf("error: %#v", err)
		os.Exit(1)
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		go func() {
			if externalIPChecker.HasNewExternalIP() {
				api.UpdateDNSRecord(
					context.Background(),
					os.Getenv("CF_ZONE_ID"),
					os.Getenv("CF_RECORD_ID"),
					cloudflare.DNSRecord{
						Content: externalIPChecker.ExternalIP,
					},
				)
			}
		}()
	}
}
