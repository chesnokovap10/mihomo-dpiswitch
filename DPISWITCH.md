# mihomo for DPI Switch

A fork of [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo), the source of the DPI Switch
core: the cut below and every dependency, so the core builds even if the upstream repository, or any of
its dependencies, goes away.

| | |
|---|---|
| Upstream commit | `63bd52ec794b7051569b76ede2f6cdbf4c091fda` (Alpha, 27.09.2026), merged; history included |
| Changes | `adapter/parser.go` keeps only the `wireguard` outbound; `listener/parse.go` only the `socks` and `tun` inbounds; the WireGuard outbound reads the stack one packet at a time and is up before its first dial; the API keeps only what DPI Switch calls (see below) |
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

This fork carried a workaround of its own (`tolerateICMP`, reading past such an error) until
`mipstack` fixed it upstream in `3ec3a765` (29.09.2026): an unconnected socket no longer reports
asynchronous ICMP errors, as Linux does not without `IP_RECVERR`. The fork takes that `mipstack` ahead
of mihomo's own `go.mod`, and the workaround is gone. `TestWireGuardUDPSurvivesICMP` stays: after the
error, the next datagram is what both reads return -- an update of `mipstack` that brought the old
behaviour back fails it.

## The first packets through a WireGuard outbound

The first dial after a core start is the groups' health check, and its first packets are DNS queries
through the tunnel (`remote-dns-resolve`), one to each of the tunnel's servers, IPv4 and IPv6, a
few milliseconds before the handshake. On about half the starts neither was ever answered: the
handshake came back in 70 ms, and every name through the tunnel then waited for the resolver's retry
5 s later. The groups marked the tunnel dead meanwhile, and what goes through it went direct.

Traced inside the device (30.09.2026, 12 starts): the slow starts were exactly those where
`RoutineReadFromTUN` took both queries from the stack in one read, and so staged them as one
container; the fast ones, where it took them one by one. The container was encrypted and written to
the socket with no error, and nothing came back for it; the same queries sent one by one on the retry
were answered in 70 ms. Why the server answers neither of the pair is not visible from here.

The device now reads the stack one packet at a time (`ipStackWireguardDevice.BatchSize` is 1): 10
starts of 10 were answered at once. It costs nothing on the wire: the bind writes one datagram per
call anyway (`ClientBind.BatchSize` is 1).

`init0` also calls the device's `Up` itself, after `Start`. Starting the stack's device only queues
`EventUp`, and the event goroutine starts the peers later; a packet the first dial wrote before that
would meet a peer not yet running and be dropped. It was not seen to happen, but nothing ordered the
two.

## The API

DPI Switch runs the core as SYSTEM, and the account the service is installed for -- which need not be
an administrator -- reads the API's secret: its UI lists the connections and closes them. With the
whole API, that secret replaced the running config (`PUT /configs` with a payload), and through it had
SYSTEM download and write files under the home directory and open listeners; `/upgrade` swapped the
binary for an upstream release, outside the service's watch.

The API keeps what DPI Switch calls (`hub/route/dpiswitch.go`):

- `PUT /configs` re-reads the core's own config file only: no payload, no other path;
- `PATCH /configs` only switches TUN off (`{"tun":{"enable":false}}`), as the service does before it
  stops the core;
- `/upgrade`, `/restart` and `/configs/geo` are gone.

Both writes hand their handler a body of their own making, so nothing else a request carried reaches
it. `TestDPISwitchAPI` sends what is refused, `TestDPISwitchAPIPasses` what is taken.

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
