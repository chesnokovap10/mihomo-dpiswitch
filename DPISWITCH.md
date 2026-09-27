# mihomo for DPI Switch

A private copy of [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo), the source of the DPI Switch
core: the cut below, one fix of its own, and every dependency, so the core builds even if the upstream
repository, or any of its dependencies, goes away.

| | |
|---|---|
| Upstream commit | `f103639c808d93a2c34cae56757b458862871b22` (the one DPI Switch is tested with), history included |
| Changes | `adapter/parser.go` keeps only the `wireguard` outbound; `listener/parse.go` only the `socks` and `tun` inbounds |
| Fix | `adapter/outbound/wireguard.go`: a peer's ICMP error no longer ends a UDP session through a WireGuard outbound (see below) |
| Dependencies | all in `vendor/`: the build needs no network |
| License | GPL-3.0, as upstream (`LICENSE`) |

## Build

```powershell
$env:CGO_ENABLED = "0"
go build -mod=vendor -trimpath -tags no_tailscale,no_zerotier,no_easytier,no_fake_tcp -ldflags "-s -w" -o mihomo.exe .
```

These are the same flags as DPI Switch's `tools\build-mihomo.ps1`.

## The fix: UDP through WireGuard and ICMP errors

The WireGuard outbound runs on mihomo's own `mips` IP stack. Like Linux, `mipstack` hands an
unconnected UDP socket the ICMP error of any target it sent to recently, as the result of its next
read. The tunnel takes any read error as the end of the UDP session (`tunnel.handleUDPToLocal`): it
closes the socket and drops the NAT entry, and the next packet opens a new one on another port.

One socket talking to many peers -- BitTorrent's uTP and DHT -- meets an unreachable peer every few
seconds, and lost every other flow each time. Measured in DPI Switch (27.09.2026), the same torrent
through the same server: the AmneziaWG client 17.5 MB/s, ~15 of it over uTP; mihomo's WireGuard
outbound 0.5 MB/s, TCP only, nothing received over UDP. Over DIRECT uTP worked: Go's own UDP sockets
do not report these errors on read.

`tolerateICMP` wraps the outbound's UDP socket and reads past such an error. Leaving the errors queued
with `SetReceiveErrors` is no fix: the error queue shares the receive capacity, and unread errors would
crowd the datagrams out. `TestWireGuardUDPSurvivesICMP` shows the error on a bare socket and the
datagram through the wrapper.

## Using it from DPI Switch

`tools\build-mihomo.ps1` builds from this repository, at the commit it pins. With no network for the
dependencies:

```powershell
$env:GOFLAGS = "-mod=vendor"; $env:GOPROXY = "off"
.\tools\build-mihomo.ps1
```

The upstream tags in the pinned commit's history are here too: without them Go stamps the build
`v0.0.0-…` instead of `v1.19.32-…`. Cutting an already-cut tree changes nothing.

## Updating to a newer upstream

```powershell
git remote add upstream https://github.com/MetaCubeX/mihomo.git
git fetch upstream
git rebase <new commit>        # the cut commit replays onto it
go mod vendor
```

The cut and the fix replay onto it. Then commit and build, and test the core in DPI Switch before
moving its pinned commit.
