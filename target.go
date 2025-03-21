package awsping

import (
	"fmt"
	"net"
)

// Targetter is an interface to get target's IP or URL
type Targetter interface {
	GetURL() string
	GetIP() (*net.TCPAddr, error)
}

// AWSTarget implements Targetter for AWS
type AWSTarget struct {
	HTTPS   bool
	Code    string
	Service string
	Rnd     string
}

func GetTld(code string) string {
	tld := "amazonaws.com"
	if code[:2] == "cn" {
		tld = "amazonaws.com.cn"
	}
	return tld
}

// GetURL return URL for AWS target
func (r *AWSTarget) GetURL() string {
	proto := "http"
	if r.HTTPS {
		proto = "https"
	}
	hostname := fmt.Sprintf("%s.%s.%s", r.Service, r.Code, GetTld(r.Code))
	url := fmt.Sprintf("%s://%s/ping?x=%s", proto, hostname, r.Rnd)
	return url
}

// GetIP return IP for AWS target
func (r *AWSTarget) GetIP() (*net.TCPAddr, error) {
	tcpURI := fmt.Sprintf("%s.%s.%s:443", r.Service, r.Code, GetTld(r.Code))
	return net.ResolveTCPAddr("tcp4", tcpURI)
}
