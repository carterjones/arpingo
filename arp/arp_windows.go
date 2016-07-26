package arp

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/sys/windows"
	"net"
	"syscall"
	"unsafe"
)

var (
	iphlpapi          = windows.NewLazySystemDLL("Iphlpapi.dll")
	getIpNetTableProc = iphlpapi.NewProc("GetIpNetTable")
)

const (
	maxLenPhysAddr          int = 8
	noError                 int = 0
	errorInsufficientBuffer int = 0x7A
	errorInvalidParameter   int = 0x57
	errorNoData             int = 0xE8
	errorNotSupported       int = 0x32
)

type mib_IPNETROW struct { // 24 bytes
	Index       int32                // DWORD (4 bytes)
	PhysAddrLen int32                // DWORD (4 bytes)
	PhysAddr    [maxLenPhysAddr]byte // 8-byte array
	Addr        int32                // DWORD (4 bytes)
	Type        int32                // DWORD (4 bytes)
}

func (r mib_IPNETROW) IpAddr() net.IP {
	ipBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ipBytes, uint32(r.Addr))
	return net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
}

func (r mib_IPNETROW) MacAddress() net.HardwareAddr {
	var ha net.HardwareAddr
	for i := int32(0); i < r.PhysAddrLen; i++ {
		ha = append(ha, r.PhysAddr[i])
	}
	return ha
}

func (r mib_IPNETROW) TypeString() string {
	switch r.Type {
	case int32(Other):
		return "other"
	case int32(Dynamic):
		return "dynamic"
	case int32(Static):
		return "static"
	case int32(Invalid):
		return "invalid"
	default:
		return "unknown type"
	}
}

// On 64-bit systems in Windows, the default uninitialized value of this struct is 12 bytes long:
//   - dwNumEntries is 4 bytes because it's a DWORD
//   - table is 8 bytes because it's a pointer to an array
// Because the implementation of GetIpNetTable assumes an inline array will be made in the
// MIB_IPNETTABLE struct, and golang structs require fixed-length arrays when initializing them XOR
// to use a slice (as we do here) that points to another portion of memory, we have to marshal the
// table from a raw byte array.
type mib_IPNETTABLE struct {
	dwNumEntries int32          // 4 bytes
	table        []mib_IPNETROW // any size
}

func getIpNetTable() (ipNetTable mib_IPNETTABLE, err error) {
	// DWORD GetIpNetTable(
	//   _Out_   PMIB_IPNETTABLE pIpNetTable,
	//   _Inout_ PULONG          pdwSize,
	//   _In_    BOOL            bOrder
	// );

	dwSize := int32(0)
	bOrder := 1 // true

	// Run once to get the required buffer size.
	var numArgs uintptr = 3
	ret, _, err := syscall.Syscall(getIpNetTableProc.Addr(),
		numArgs,
		uintptr(unsafe.Pointer(&ipNetTable)),
		uintptr(unsafe.Pointer(&dwSize)),
		uintptr(bOrder))

	// We expect an insufficient buffer size. If that is not encountered, throw an error.
	if ret != uintptr(errorInsufficientBuffer) {
		return ipNetTable, err
	}

	// Prepare a buffer to receive the table.
	buffer := make([]byte, dwSize)

	// Call it again to receive the table.
	ret, _, err = syscall.Syscall(getIpNetTableProc.Addr(),
		numArgs,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&dwSize)),
		uintptr(bOrder))

	// If neither NO_ERROR or ERROR_NO_DATA occurs, then report the error.
	if ret != uintptr(noError) && ret != uintptr(errorNoData) {
		return ipNetTable, err
	}

	// The first four bytes hold the number of rows.
	numEntries := int32(binary.LittleEndian.Uint32(buffer[:4]))
	ipNetTable.dwNumEntries = int32(numEntries)

	// Skip bytes 5 through 12, since that is an address to the start of the array.
	sizeofNumEntries := 4 // DWORD
	sizeofMib_IPNETROW := int32(unsafe.Sizeof(mib_IPNETROW{}))
	for i := int32(0); i < numEntries; i++ {
		// Skip the first four bytes.
		offset := int32(sizeofNumEntries)

		// Skip all rows prior to this one.
		offset = offset + (sizeofMib_IPNETROW * i)

		// Get the row's bytes.
		rowBytes := buffer[offset : offset+sizeofMib_IPNETROW]

		// Read the bytes to a struct.
		row := mib_IPNETROW{}
		buf := bytes.NewBuffer(rowBytes)
		err = binary.Read(buf, binary.LittleEndian, &row)
		if err != nil {
			return ipNetTable, nil
		}

		// Add the row to the table.
		ipNetTable.table = append(ipNetTable.table, row)
	}

	return ipNetTable, err
}

func GetArpTable() (ArpTable, error) {
	ret, err := getIpNetTable()
	if err != nil {
		return nil, err
	}

	at := ArpTable{}
	for _, v := range ret.table {
		entry := ArpEntry{
			Index:   int(v.Index),
			MacAddr: v.MacAddress(),
			IpAddr:  v.IpAddr(),
			Type:    RowType(v.Type),
		}
		at = append(at, entry)
	}

	return at, nil
}
