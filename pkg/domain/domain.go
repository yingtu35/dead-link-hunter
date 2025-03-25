package domain

import (
	"errors"
	"net/url"
	"strings"
)

// GetProtocol returns the protocol of a given URL
func GetProtocol(u string) (string, error) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return "", errors.New("error parsing URL")
	}
	return parsedUrl.Scheme, nil
}

// GetDomain returns the domain of a given URL
// Refers to https://stackoverflow.com/questions/67650694/how-do-i-retrieve-the-domain-from-a-url
func GetDomain(u string) (string, error) {
	var hostname string
	var temp []string

	parsedUrl, err := url.Parse(u)
	if err != nil {
		return "", errors.New("error parsing URL")
	}
	var urlstr string = parsedUrl.String()

	if strings.HasPrefix(urlstr, "https") {
		hostname = strings.TrimPrefix(urlstr, "https://")
	} else if strings.HasPrefix(urlstr, "http") {
		hostname = strings.TrimPrefix(urlstr, "http://")
	} else {
		hostname = urlstr
	}

	if strings.HasPrefix(hostname, "www") {
		hostname = strings.TrimPrefix(hostname, "www.")
	}
	if strings.Contains(hostname, "/") {
		temp = strings.Split(hostname, "/")
		hostname = temp[0]
	}
	return hostname, nil
}

func IsSameDomain(domain string, u string) bool {
	d, err := GetDomain(u)
	return err == nil && domain == d
}
