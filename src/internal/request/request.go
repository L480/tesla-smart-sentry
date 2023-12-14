package request

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type Endpoint struct {
	Url                string
	Method             string
	Headers            map[string]string
	Body               string
	InsecureSkipVerify bool
}

func Connect(e Endpoint) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: e.InsecureSkipVerify},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest(e.Method, e.Url, strings.NewReader(e.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint: %s", err)
	}

	for k, v := range e.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %s", err)
	}
	return resp, nil
}

func SubscribeSse(r *http.Response, c func([]byte)) error {
	for {
		data := make([]byte, 1024)
		_, err := r.Body.Read(data)
		if err != nil {
			return fmt.Errorf("failed to retrieve SSE data: %s", err)
		}
		re := regexp.MustCompile("^[^[]*|\n*")
		data = []byte(re.ReplaceAllString(string(data), ""))
		data = bytes.Trim(data, "\x00")
		c(data)
	}
}
