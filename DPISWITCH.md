# mihomo for DPI Switch

A private copy of [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo), the source of the DPI Switch
core: the cut below and every dependency, so the core builds even if the upstream repository, or any of
its dependencies, goes away.

| | |
|---|---|
| Upstream commit | `63bd52ec794b7051569b76ede2f6cdbf4c091fda` (Alpha, 27.09.2026), merged; history included |
| Changes | `adapter/parser.go` keeps only the `wireguard` outbound; `listener/parse.go` only the `socks` and `tun` inbounds |
| mipstack | `3ec3a765c58a` (29.09.2026), ahead of upstream's: it fixes UDP through WireGuard (see below) |
| Dependencies | all in `vendor/`: the build needs no network |
| License | GPL-3.0, as upstream (`LICENSE`) |

## Build

```powershell
$env:CGO_ENABLED = "0"
go build -mod=vendor -trimpath -tags no_tailscale,no_zerotier,no_easytier,no_fake_tcp -ldflags "-s -w" -o mihomo.exe .
```

These are the same flags as DPI Switch's `tools\build-mihomo.ps1`.

## UDP through WireGuard and ICMP errors

The WireGuard outbound runs on mihomo's own `mips` IP stack. Up to 26.09.2026 `mipstack` handed an
unconnected UDP socket the ICMP error of any target it sent to recently, as the result of its next
read, and the tunnel took any read error as the end of the UDP session (`tunnel.handleUDPToLocal`): it
closed the socket and dropped the NAT entry, and the next packet opened a new one on another port.

One socket talking to many peers -- BitTorrent's uTP and DHT -- met an unreachable peer every few
seconds, and lost every other flow each time. Measured in DPI Switch (27.09.2026), the same torrent
through the same server: the AmneziaWG client 17.5 MB/s, ~15 of it over uTP; mihomo's WireGuard
outbound 0.5 MB/s, TCP only, nothing received over UDP.

This copy carried a workaround of its own (`tolerateICMP`, reading past such an error) until
`mipstack` fixed it upstream in `3ec3a765` (29.09.2026): an unconnected socket no longer reports
asynchronous ICMP errors, as Linux does not without `IP_RECVERR`. The copy takes that `mipstack` ahead
of mihomo's own `go.mod`, and the workaround is gone. `TestWireGuardUDPSurvivesICMP` stays: after the
error, the next datagram is what both reads return -- an update of `mipstack` that brought the old
behaviour back fails it.

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
