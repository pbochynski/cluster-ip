package ip

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type providerResponse struct {
	err  error
	name string
	ip   string
	res  http.Response
	url  string
}

type IPService struct {
	url      string
	name     string
	jsonPath string
	client   *http.Client
}

func NewIPService(name, url string, timeout time.Duration) *IPService {
	return &IPService{
		url:    url,
		name:   name,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *IPService) Check() providerResponse {
	var result map[string]any
	nt := NewIPService(p.name, p.url, time.Second*5)
	res, err := nt.client.Get(p.url)
	if err != nil {
		return providerResponse{err: err}
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return providerResponse{err: err}
	}
	json.Unmarshal(body, &result)
	defer res.Body.Close()
	ip := result[p.jsonPath]

	return providerResponse{name: p.name, res: *res, url: p.url, ip: ip.(string)}
}

func (p *IPService) Name() string {
	return p.name
}

func IsValidIP4(ipAddress string) bool {
	ipAddress = strings.Trim(ipAddress, " ")
	re, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	return re.MatchString(ipAddress)
}

// worker defines our worker func. as long as there is a job in the
// "queue" we continue to pick up  the "next" job
func worker(jobs <-chan IPService, results chan<- providerResponse) {
	for n := range jobs {
		results <- n.Check()
	}
}

var (
	providers = []IPService{
		{
			name:     "ipwho.is",
			url:      "https://ipwho.is",
			jsonPath: "ip",
		},
		{
			name:     "jsonip.com",
			url:      "https://jsonip.com",
			jsonPath: "ip",
		},
		{
			name:     "ifconfig.me",
			url:      "https://ifconfig.me/all.json",
			jsonPath: "ip_addr",
		},
		{
			name:     "ipinfo.io",
			url:      "https://ipinfo.io/json",
			jsonPath: "ip",
		},
	}
)

func GetIP(min int) (string, error) {
	// Make buffered channels
	buffer := len(providers)
	jobsPipe := make(chan IPService, buffer)           // Jobs will be of type `Tap`
	resultsPipe := make(chan providerResponse, buffer) // Results will be of type `testerResponse`

	for i := 0; i < buffer; i++ {
		go worker(jobsPipe, resultsPipe)
	}

	for _, p := range providers {
		jobsPipe <- p
	}

	var ip string
	counter := 0
	providers := []string{}
	for i := 0; i < buffer && counter < min; i++ {
		r := <-resultsPipe
		if IsValidIP4(r.ip) {
			counter++
			providers = append(providers, r.name)
			if ip == "" {
				ip = r.ip
			} else if ip != r.ip {
				return "", fmt.Errorf("got 2 different IPs: %s, %s", ip, r.ip)
			}
		}

	}
	if counter < min {
		return "", fmt.Errorf("only %d of %d required services returned valid IP", counter, min)
	}
	return ip, nil
}
