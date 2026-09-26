package listener

import (
	"fmt"

	"github.com/metacubex/mihomo/common/structure"
	C "github.com/metacubex/mihomo/constant"
	IN "github.com/metacubex/mihomo/listener/inbound"
)

func ParseListener(mapping map[string]any) (C.InboundListener, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "inbound", WeaklyTypedInput: true, KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}

	var (
		listener C.InboundListener
		err      error
	)
	switch proxyType {
	case "socks":
		socksOption := &IN.SocksOption{UDP: true}
		err = decoder.Decode(mapping, socksOption)
		if err != nil {
			return nil, err
		}
		listener, err = IN.NewSocks(socksOption)
	case "tun":
		tunOption := &IN.TunOption{
			Stack:     C.TunGvisor,
			DNSHijack: []string{"0.0.0.0:53"}, // default hijack all dns query
		}
		err = decoder.Decode(mapping, tunOption)
		if err != nil {
			return nil, err
		}
		listener, err = IN.NewTun(tunOption)
	default:
		return nil, fmt.Errorf("unsupport proxy type: %s", proxyType)
	}
	return listener, err
}
