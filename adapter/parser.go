package adapter

import (
	"fmt"

	"github.com/metacubex/mihomo/adapter/outbound"
	"github.com/metacubex/mihomo/common/structure"
	C "github.com/metacubex/mihomo/constant"
)

func ParseProxy(mapping map[string]any, options ...ProxyOption) (C.Proxy, error) {
	decoder := structure.NewDecoder(structure.Option{TagName: "proxy", WeaklyTypedInput: true, KeyReplacer: structure.DefaultKeyReplacer})
	proxyType, existType := mapping["type"].(string)
	if !existType {
		return nil, fmt.Errorf("missing type")
	}

	opt := applyProxyOptions(options...)
	basicOption := outbound.BasicOption{
		DialerForAPI: opt.DialerForAPI,
		TunnelForAPI: opt.TunnelForAPI,
		ProviderName: opt.ProviderName,
	}

	var (
		proxy outbound.ProxyAdapter
		err   error
	)
	switch proxyType {
	case "wireguard":
		wgOption := &outbound.WireGuardOption{BasicOption: basicOption}
		err = decoder.Decode(mapping, wgOption)
		if err != nil {
			break
		}
		proxy, err = outbound.NewWireGuard(*wgOption)
	default:
		return nil, fmt.Errorf("unsupport proxy type: %s", proxyType)
	}

	if err != nil {
		return nil, err
	}

	if muxMapping, muxExist := mapping["smux"].(map[string]any); muxExist {
		muxOption := &outbound.SingMuxOption{}
		err = decoder.Decode(muxMapping, muxOption)
		if err != nil {
			return nil, err
		}
		if muxOption.Enabled {
			proxy, err = outbound.NewSingMux(*muxOption, proxy)
			if err != nil {
				return nil, err
			}
		}
	}

	proxy = outbound.NewAutoCloseProxyAdapter(proxy)
	return NewProxy(proxy), nil
}

type proxyOption struct {
	DialerForAPI C.Dialer
	TunnelForAPI C.Tunnel
	ProviderName string
}

func applyProxyOptions(options ...ProxyOption) proxyOption {
	opt := proxyOption{}
	for _, o := range options {
		o(&opt)
	}
	return opt
}

type ProxyOption func(opt *proxyOption)

func WithDialerForAPI(dialer C.Dialer) ProxyOption {
	return func(opt *proxyOption) {
		opt.DialerForAPI = dialer
	}
}

func WithTunnelForAPI(tunnel C.Tunnel) ProxyOption {
	return func(opt *proxyOption) {
		opt.TunnelForAPI = tunnel
	}
}

func WithProviderName(name string) ProxyOption {
	return func(opt *proxyOption) {
		opt.ProviderName = name
	}
}
