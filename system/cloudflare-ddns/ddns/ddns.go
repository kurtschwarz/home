package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func boolPtr(v bool) *bool {
	return &v
}

var cloudflare *CloudflareClient

type CloudflareClient struct {
	client http.Client
}

func (c *CloudflareClient) CheckRecordIP(zone string, name string, expected *net.IP) (recordID *string, _ *bool, err error) {
	var req *http.Request
	if req, err = http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s", zone, name), nil); err != nil {
		return nil, nil, err
	}

	var getRecordResponse = struct {
		Result []struct {
			Id      string `json:"id"`
			Content string `json:"content"`
		} `json:"result"`
	}{}

	if err = c.Do(req, &getRecordResponse); err != nil {
		return nil, nil, err
	}

	for _, result := range getRecordResponse.Result {
		if net.ParseIP(result.Content).Equal(*expected) {
			return &result.Id, boolPtr(false), nil
		}
	}

	if len(getRecordResponse.Result) > 0 {
		return &getRecordResponse.Result[0].Id, boolPtr(false), nil
	}

	return nil, boolPtr(false), nil
}

func (c *CloudflareClient) UpdateRecord(zone string, id string, name string, content string) (updated bool, err error) {
	var req *http.Request
	if req, err = http.NewRequest("PUT", fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone, id), bytes.NewBuffer([]byte(fmt.Sprintf(`{"type":"A","name":"%s","content":"%s"}`, name, content)))); err != nil {
		return false, err
	}

	var updateRecordResponse = struct {
		Success bool `json:"success"`
	}{}

	if err = c.Do(req, &updateRecordResponse); err != nil {
		return false, err
	}

	return true, nil
}

func (c *CloudflareClient) Do(req *http.Request, i interface{}) (err error) {
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", os.Getenv("CF_API_TOKEN")))
	req.Header.Add("content-type", "application/json")

	var res *http.Response
	if res, err = c.client.Do(req); err != nil {
		return err
	}

	var body []byte
	if body, err = ioutil.ReadAll(res.Body); err != nil {
		return err
	}

	json.Unmarshal(body, &i)

	return nil
}

type Config struct {
	Zones map[string]struct {
		Records []string `json:"records"`
	} `json:"zones"`
}

func getConfig() (config *Config, err error) {
	var file *os.File
	if file, err = os.Open("/ddns.json"); err != nil {
		return nil, err
	}

	var bytes []byte
	if bytes, err = ioutil.ReadAll(file); err != nil {
		return nil, err
	}

	config = &Config{}
	if err = json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	return config, nil
}

func getPublicIPv4Address() (*net.IP, error) {
	var res *http.Response
	var err error

	if res, err = http.Get("https://checkip.amazonaws.com/"); err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get public ipv4 address, checkip.amazonaws.com returned %s (expected 200 OK)", res.Status)
	}

	var body []byte
	if body, err = ioutil.ReadAll(res.Body); err != nil {
		return nil, err
	}

	ipv4Address := net.ParseIP(strings.TrimSpace(string(body)))

	log.Printf("==> External IPv4: %s\n", ipv4Address)

	return &ipv4Address, nil
}

func init() {
	cloudflare = &CloudflareClient{}
}

func main() {
	var err error

	var config *Config
	if config, err = getConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}

	var publicIPv4Address *net.IP
	if publicIPv4Address, err = getPublicIPv4Address(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}

	for zoneID, zone := range config.Zones {
		log.Printf("Checking Records in Zone (%s)", zoneID)

		for _, record := range zone.Records {
			log.Printf("Checking Record (%s) in Zone (%s)", record, zoneID)

			var recordID *string
			var matched *bool
			if recordID, matched, err = cloudflare.CheckRecordIP(zoneID, record, publicIPv4Address); err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				os.Exit(1)
			}

			if !*matched {
				log.Printf("==> Record needs to be updated to %s (ID %s)", publicIPv4Address, *recordID)

				if _, err = cloudflare.UpdateRecord(zoneID, *recordID, record, publicIPv4Address.To4().String()); err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					os.Exit(1)
				}

				log.Printf("==> Record (%s) updated to %s", *recordID, publicIPv4Address)
			}

			log.Printf("==> Skipping, record does not need to be updated")
		}
	}

	os.Exit(0)
}
