package network

import (
	"net"
	"testing"
)

func TestAllocate(t *testing.T) {
	IP, ipnet, _ := net.ParseCIDR("192.168.0.1/24")
	t.Logf("gateway: %v", IP)
	ip, _ := ipAllocator.Allocate(ipnet)
	t.Logf("alloc ip: %v", ip)
}

func TestRelease(t *testing.T) {
	ip, ipnet, _ := net.ParseCIDR("192.168.0.5/24")
	ipAllocator.Release(ipnet, &ip)
	t.Logf("release ip: %v", ip)
}
