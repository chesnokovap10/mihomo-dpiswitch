package packetaddr

import (
	E "github.com/metacubex/sing/common/exceptions"
	M "github.com/metacubex/sing/common/metadata"
)

const SeqPacketMagicAddress = "sp.packet-addr.v2fly.arpa"

var AddressSerializer = M.NewSerializer(
	M.AddressFamilyByte(0x01, M.AddressFamilyIPv4),
	M.AddressFamilyByte(0x02, M.AddressFamilyIPv6),
)

var ErrFqdnUnsupported = E.New("packetaddr: fqdn unsupported")
