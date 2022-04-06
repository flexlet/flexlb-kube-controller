package utils

import (
	"bytes"
	"fmt"
	"net"
)

// cidr contai
func IpInNetwork(ipstr string, cidr string) bool {
	ip := net.ParseIP(ipstr)
	if _, ipnet, err := net.ParseCIDR(cidr); err != nil {
		return false
	} else {
		return ipnet.Contains(ip)
	}
}

// allocate ip from cidr
func AllocIPFromCidr(cidr string, exist []string) (*string, error) {
	cidrIp, cidrNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	for ip := cidrIp.Mask(cidrNet.Mask); cidrNet.Contains(ip); inc(ip) {
		ipstr := ip.String()
		if !ListContains(exist, ipstr) {
			return &ipstr, nil
		}
	}
	return nil, fmt.Errorf("cidr is full")
}

// allocate ip from range
func AllocIPFromRange(start string, end string, exist []string) (*string, error) {
	startIp := net.ParseIP(start)
	endIp := net.ParseIP(end)

	for ip := startIp; bytes.Compare(ip, endIp) < 0; inc(ip) {
		ipstr := ip.String()
		if !ListContains(exist, ipstr) {
			return &ipstr, nil
		}
	}
	return nil, fmt.Errorf("ip range is full")
}

// inc ip
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
