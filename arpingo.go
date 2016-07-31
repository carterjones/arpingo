package main

import (
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// Based off of https://gist.github.com/kotakanbe/d3059af990252ba89a82
func getIpsInCidr(cidr string) ([]net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	// http://play.golang.org/p/m8TNTtygK0
	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	var ips []net.IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip)
	}

	// Remove network address and broadcast address.
	return ips, nil
}

func pingIp(ip net.IP) (success bool, err error) {
	// Create the echo message.
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{1, 1, []byte("hello")},
	}

	// Convert the message into raw bytes.
	wb, err := wm.Marshal(nil)
	if err != nil {
		return false, err
	}

	// Prepare to send the message to the destination IP.
	c, err := icmp.ListenPacket("ip4:icmp", "127.0.0.1")
	if err != nil {
		return false, err
	}

	// Create a destination IPAddr variable.
	ipaddr, err := net.ResolveIPAddr("ip4", ip.String())
	if err != nil {
		return false, err
	}

	// Send the message to the destination IP.
	if n, err := c.WriteTo(wb, ipaddr); err != nil {
		return false, err
	} else if n != len(wb) {
		return false, fmt.Errorf("got %v; want %v", n, len(wb))
	} else {
		return true, nil
	}
}

func main() {
	// Verify that a parameter was passed in via the command line.
	if len(os.Args) < 2 {
		progName := filepath.Base(os.Args[0])
		fmt.Printf("usage: %v <cidr>\n", progName)
		return
	}

	verbose := false

	// Set the CIDR variable.
	cidr := os.Args[1]

	// Get a list of all the IPs in the CIDR.
	ips, err := getIpsInCidr(cidr)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Ping all the IPs concurrently.
	var wg sync.WaitGroup
	wg.Add(len(ips))
	for _, ip := range ips {
		go func() {
			defer wg.Done()
			_, err = pingIp(ip)
			if err != nil {
				if verbose {
					fmt.Println(err)
				}
			}
		}()
	}
	wg.Wait()

	// TODO: get the data from the arp table in an OS-agnostic way.
}
