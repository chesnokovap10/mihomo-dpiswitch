# mihomo for DPI Switch

A private copy of [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo), kept so the DPI Switch core
can be built even if the upstream repository, or any of its dependencies, goes away.

| | |
|---|---|
| Upstream commit | `f103639c808d93a2c34cae56757b458862871b22` (the one DPI Switch is tested with), history included |
| Changes | `adapter/parser.go` keeps only the `wireguard` outbound; `listener/parse.go` only the `socks` and `tun` inbounds |
| Dependencies | all in `vendor/`: the build needs no network |
| License | GPL-3.0, as upstream (`LICENSE`) |

## Build

```powershell
$env:CGO_ENABLED = "0"
go build -mod=vendor -trimpath -tags no_tailscale,no_zerotier,no_easytier,no_fake_tcp -ldflags "-s -w" -o mihomo.exe .
```

These are the same flags as DPI Switch's `tools\build-mihomo.ps1`. Built from this tree without the
vendor directory, the result is byte-identical to the core DPI Switch 1.3.0 embeds. With it, the code
is the same, and only the module checksums in the build info differ.

## Using it from DPI Switch

`tools\build-mihomo.ps1` clones upstream by default. Both cases below were checked:

```powershell
# upstream is gone, the module proxy still has the dependencies:
# the same pinned commit from here, the same core byte for byte
.\tools\build-mihomo.ps1 -Repo https://github.com/chesnokovap10/mihomo-dpiswitch.git

# the dependencies are gone too: this branch's head, vendor/, no network
$env:GOFLAGS = "-mod=vendor"; $env:GOPROXY = "off"
.\tools\build-mihomo.ps1 -Repo https://github.com/chesnokovap10/mihomo-dpiswitch.git -Commit <head of main>
```

The upstream tags in the pinned commit's history are here too. Without them Go stamps the build
`v0.0.0-…` instead of `v1.19.32-…`, and the core is no longer byte-identical. Cutting an already-cut
tree changes nothing.

## Updating to a newer upstream

```powershell
git remote add upstream https://github.com/MetaCubeX/mihomo.git
git fetch upstream
git rebase <new commit>        # the cut commit replays onto it
go mod vendor
```

Then commit and build. Test the core in DPI Switch before moving its pinned commit.
