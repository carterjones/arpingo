package arp

import (
	"net"
)

type RowType int

const (
	_             = iota
	Other RowType = iota
	Invalid
	Dynamic
	Static
)

type ArpEntry struct {
	Index   int
	MacAddr net.HardwareAddr
	IpAddr  net.IP
	Type    RowType
}

func (e ArpEntry) String() string {
	return e.MacAddr.String() + ": " + e.IpAddr.String()
}

type ArpTable []ArpEntry
