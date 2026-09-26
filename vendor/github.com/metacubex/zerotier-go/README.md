# zerotier-go

`zerotier-go` is a pure Go translation of the platform-independent ZeroTier
One node engine. It targets Go 1.20, uses no cgo, and leaves sockets,
persistent storage, and virtual network devices to the embedding application.

The current compatibility baseline is ZeroTierOne `dev` commit
`899352e38405968516bb12a770f0ac02f6058fa8`. The public API is not stable yet.

## Scope and compatibility

The package implements the client-side core needed by an embedded leaf node:

- identities, packet cryptography, fragmentation, WHOIS, ECHO, RENDEZVOUS,
  authenticated path discovery, peer/path caching, moons, and private planets;
- controller configuration, managed addresses, routes, DNS, membership,
  ownership, tags, capabilities, revocations, and the ZeroTier rules engine;
- FRAME, EXT_FRAME, multicast LIKE/GATHER and sender/replicator forwarding,
  active-bridge route learning, and access to hosts behind remote bridges;
- trusted paths, physical MTU policy, memorized endpoints, path blacklists,
  protocol 12 AES-GMAC-SIV, protocol 13 encrypted HELLO, remote trace, user
  messages, QoS measurements, and the official bonding policies;
- controllerless `0xff` ad-hoc networks and the official TCP fallback record
  format, fail-forward policy, and established session lifecycle in the
  `tcpfallback` package.

Wire compatibility is the goal, but this is not a drop-in replacement for the
official daemon or its service API.

### Resource hardening

Wire-format maxima remain aligned with the official core, including 64 paths
per peer, 32 receive/transmit queue entries, 128 routes and capabilities, 512
specialists, and 65,536 bond flows. A few process-wide caches are deliberately
smaller or explicitly bounded because an embedded client cannot assume daemon-
scale memory:

- learned non-root peers are capped at 4,096; the official peer table has no
  comparable fixed count;
- learned active-bridge routes are capped at 65,536 instead of the official
  `ZT_MAX_BRIDGE_ROUTES` value of 67,108,864;
- multicast groups and learned members are capped at 4,096 each, with at most
  32 pending receive packets, 32 pending multicasts per group, and 256 pending
  frame-related transmissions in total;
- controller configuration assembly is capped at 1 MiB, and deferred
  credential/capability work has separate small bounded queues.

When a limit is reached the node drops or evicts cacheable work; it does not
change authenticated wire formats or controller-advertised protocol maxima.

## Wire protocol versions

The node advertises ZeroTier wire protocol `13`, accepts HELLO negotiation from
protocol `4` onward, and advertises node compatibility version `1.16.2`. The
node version is peer/controller metadata; it is not a claim that this package
implements every daemon or service feature shipped in ZeroTier One 1.16.2.

The minimum of 4 is inherited from the current official core, not an omission
chosen by this translation. Protocols 1-3 cover the pre-Hashcash identity
formats, two older multicast designs, and a protocol-3 key-agreement/cipher
replacement. Protocol 4 introduced another breaking identity-format change.
Supporting older peers would therefore require parallel legacy identity,
key-agreement, cipher, HELLO, and multicast implementations; lowering
`ProtocolVersionMin` alone would only admit packets that the current identity
and crypto code cannot interpret. Current ZeroTier One rejects them as well.

Outgoing HELLO packets always advertise protocol 13. The received version is
stored per peer and selects only the compatibility behavior that actually
differs by version:

| Peer protocol | Official ZeroTier One releases | Support in this package |
| --- | --- | --- |
| below 4 | 0.2.0 through early 0.6.0 | Rejected for the incompatible historical identity, crypto, and multicast formats described above. |
| 4 | 0.6.0 through 1.0.6 | Accepted. Path confirmation uses a full HELLO instead of ECHO. |
| 5 | 1.1.0 through 1.1.5 | ECHO path probes and in-band planet/moon updates are implemented. The known ZeroTier One 1.1.0 ECHO exception still uses full HELLO. |
| 6 | 1.1.5 through 1.1.10 | Binary network-configuration values and the corresponding legacy/current configuration parsers are implemented. |
| 7 | 1.1.10 through 1.1.17 | Configured trusted physical paths and their cipher marker are implemented. |
| 8 | 1.1.17 through 1.2.0 | Multipart network configurations, tags, capabilities, and modern certificate-of-membership delivery are implemented. |
| 9 | 1.2.0 through 1.2.14 | Accepted. The official protocol history lists no separate wire-format feature for this bump. |
| 10 | 1.4.0 through 1.4.6 | Accepted. The official protocol history lists no separate wire-format feature for this bump. |
| 11 | 1.4.7 through 1.4.8 | Multipath negotiation, QoS measurement, and the supported load-balancing policies are implemented. |
| 12 | 1.4.8 through 1.16.0 | AES-GMAC-SIV is used for packets that require payload encryption; authenticated cleartext control packets retain legacy armor. |
| 13 | 1.16.0 onward | Extended, ephemeral encrypted HELLO armor is implemented. Receiving it is automatic; sending it requires `NodeConfig.EncryptedHello` because it occurs before peer-version discovery. |
| above 13 | future | A structurally valid HELLO is accepted for forward tolerance, but only protocol-13 packet formats, ciphers, verbs, and semantics are implemented. This is not a future-version compatibility guarantee. |

Protocols 4-11 use Salsa20/12 and Poly1305 for ordinary protected packets. The
lack of `peer.protocol` branches for versions 6-11 does not mean those features
are missing: trusted paths identify themselves with a cipher selector,
network-configuration revisions are self-describing, and multipart config,
credentials, tags, capabilities, and multipath use their own verbs and payload
formats. Only the ECHO compatibility decision and the protocol-12 cipher change
require direct per-peer version branches in the current leaf-node send path.

The packet layer implements all cipher selectors used through protocol 13:
authenticated cleartext, Salsa20/12-encrypted Poly1305, configured trusted-path
markers, and AES-GMAC-SIV. It also implements extended HELLO armor, LZ4 packet
compression, and the official fragmentation format. Unknown cipher selectors
or malformed extensions are rejected.

The leaf-node verb implementation covers HELLO, ERROR, OK, WHOIS, RENDEZVOUS,
FRAME, EXT_FRAME, ECHO, multicast LIKE/GATHER/FRAME, credentials, network
configuration requests and pushes, PUSH_DIRECT_PATHS, USER_MESSAGE,
REMOTE_TRACE, ACK, QoS measurement, and path negotiation. An inbound
NETWORK_CONFIG_REQUEST receives the official unsupported-operation response
because this package is not a controller. NOP, ACK, and unknown authenticated
verbs have their payload ignored; they do not imply support for unknown payload
semantics.

Wire protocol version and controller configuration format version are separate.
Configuration requests advertise dictionary version `7`; the parser supports
the pre-v6 legacy dictionary layout and the v6+ binary layout, ignoring unknown
dictionary keys while validating all implemented fields.

Automated tests cover packet crypto, protocol-12 and protocol-13 armor,
HELLO/OK parsing, the legacy path-probe fallback, verbs, and network
configuration formats. `ztprobe` has also been used for current official-client
interoperability, but CI does not run a matrix of archived protocol 4-12 daemon
releases. Support for those historical versions describes the implemented code
paths, not separate certification against every old release.

## Implemented functionality

The current translation includes:

- ZeroTier identities, X25519 key agreement, signatures, Salsa20/12,
  Poly1305, AES-GMAC-SIV, and protocol 13 ephemeral encrypted HELLO armor;
- the embedded default planet, caller-provided private planets, signed world
  updates, persisted moons, moon discovery, and orbit/deorbit operations;
- WHOIS, HELLO, ECHO, RENDEZVOUS, PUSH_DIRECT_PATHS, authenticated path probes,
  external-surface tracking, candidate advertisement, and per-socket path
  selection using the official latency, age, priority, and IP-scope policy;
- ZeroTier packet compression, fragmentation, out-of-order reassembly, trusted
  paths, per-prefix physical MTU overrides, physical path blacklists, and
  memorized peer endpoint hints;
- official v2 peer cache files, the separate in-memory and persisted peer
  lifetimes, stale cache cleanup, cached network recovery, and periodic
  controller refresh;
- controller network configuration requests, signed chunk assembly and fast
  propagation, managed addresses, routes, DNS servers, MTU, multicast limits,
  specialists, and private-network membership verification;
- controllerless `0xff` ad-hoc networks, including port-range IPv6 mode and the
  IPv4/IPv6 multicast-anchor mode;
- certificates of membership and ownership, credential exchange, tags,
  capabilities with verified custody chains, and targeted revocations with
  fast propagation;
- sender IP/MAC authentication and the network rules engine, including the
  supported match types, capability fallback, TEE, WATCH, REDIRECT, BREAK,
  PRIORITY, super-accept behavior, and remote filter traces;
- FRAME and EXT_FRAME delivery, remote active-bridge MAC learning, bounded
  unknown-MAC forwarding to active bridges, and access to ordinary hosts
  behind an authorized remote bridge;
- multicast LIKE/GATHER membership caches, ARP ADI groups, sender-side and
  designated-replicator forwarding, per-recipient rule evaluation, pending
  replication queues, and per-peer announcements;
- USER_MESSAGE sending and receiving plus REMOTE_TRACE generation for packet,
  path, credential, network-access, and rule diagnostics;
- local lifecycle events for peer identities, authenticated paths, deduplicated
  user-traffic direct/relay selection, and trusted external-surface changes;
- active-backup, broadcast, balance-rr, balance-xor, and balance-aware bonding,
  including failover hysteresis, primary/spare paths, negotiation, capacity and
  IP-family preferences, per-peer overrides, flow affinity, and QoS path
  measurements;
- the official TCP fallback relay record format, fail-forward policy, bounded
  queue, and established connection lifecycle in `tcpfallback`; connection
  dialing remains an embedding responsibility;
- an optional `transport` package implementing the official leaf-service
  physical socket policy with injected dialing and interface discovery;
- the `iplink` L2/L3 adapter with managed route selection, bounded ARP and NDP
  caches, timed neighbor retries, IPv4/IPv6 multicast mapping, IGMP/MLD state,
  and RFC4193/6plane direct address handling.

## Embedding lifecycle

Create one `Node` per identity, provide its I/O callbacks, join a network, and
run its maintenance loop until shutdown:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

node, err := zerotier.NewNode(zerotier.NodeConfig{
    Store: store,
    Sender: zerotier.WireSenderFunc(sendUDP),
    OnNetworkConfig: applyNetworkConfig,
    OnFrame: deliverEthernetFrame,
})
if err != nil {
    return err
}
defer node.Close()

if err := node.Join(networkID); err != nil {
    return err
}
go node.RunBackgroundTasks(ctx)
```

Pass every received ZeroTier UDP datagram to `ProcessWirePacket`, preserving the
local socket ID supplied to `WireSender.Send`. Socket IDs are opaque embedding
handles; `AnyLocalSocket` means that the core has no preferred local socket or
that the received transport has no corresponding physical socket. The
embedding may use separate IPv4 and IPv6 sockets and may replace failed sockets
without recreating the node.

Public methods are safe for concurrent use. Callbacks run synchronously after
the node lock is released, so they may call back into the node, but they should
not block packet processing. Queue expensive device or application work.
Embedding services (`StateStore`, `WireSender`, `DirectPaths`, `PathCheck`, and
`PathLookup`) run inside the serialized protocol operation and must not call
back into the same node.

`EventNetworkConfigReady` means that the core accepted an initial network
configuration, whether controller-issued or locally generated for an ad-hoc
network, or recovered from a non-ready network status. It does not mean that
an embedding has successfully applied that configuration to its local virtual
interface. It corresponds to official network status
`ZT_NETWORK_STATUS_OK` and virtual-network operation
`ZT_VIRTUAL_NETWORK_CONFIG_OPERATION_UP`.
`EventNetworkConfigChanged` corresponds to official operation
`ZT_VIRTUAL_NETWORK_CONFIG_OPERATION_CONFIG_UPDATE` for a changed
configuration on an already-ready network; embedders should report their own
usable or joined state only after their `OnNetworkConfig` work succeeds.
Calling `RefreshNetwork` after a failure first emits
`EventNetworkConfigPending`. The controller response can therefore complete
with the same failure status and still produce an observable transition,
without turning ordinary repeated responses into duplicate events. During the
retry, `Network.Status` and `Network.Authentication` continue to describe the
last authoritative controller response.

Peer path events describe state boundaries rather than packet activity.
`EventPeerPathLearned` is emitted only after an unknown endpoint completes
authenticated confirmation and corresponds to official remote-trace event
`ZT_REMOTE_TRACE_EVENT__PEER_LEARNED_NEW_PATH`. `EventPeerRouteChanged` is
emitted only after a successful user-frame send and only when its `Route`
differs from the last reported direct or relayed route. Heartbeats, probes,
receive timestamps, latency samples, QoS measurements, and direct endpoint
changes within the same route kind do not emit lifecycle events.

Event and network-configuration callbacks are serialized in the order in which
their protocol-state changes commit. They run without the node lock and may
reenter the node. A state callback produced by that reentry is queued behind
callbacks for changes that already committed and is not invoked recursively.
Frame, user-message, and remote-trace callbacks also run without the node lock,
but may run concurrently so high-volume data delivery cannot accumulate behind
a blocked state callback. Embedding callbacks should return promptly.

Event payloads keep local, peer, and reporting identities in `NodeAddress`,
`PeerAddress`, and `ReporterAddress` respectively. `PeerRole` uses the official
`ZT_PEER_ROLE_LEAF`, `ZT_PEER_ROLE_MOON`, and `ZT_PEER_ROLE_PLANET` role model;
`PeerStatus.Role` exposes the same classification in snapshots.

`RunBackgroundTasks` should have exactly one active instance. Embedders with an
existing scheduler may call `ProcessBackgroundTasks` at the returned deadline
instead.

Startup and controller failures do not poison persisted state. An embedding can
close a failed node and construct another one over the same `StateStore` when a
later connection attempt should retry.

## State and I/O ownership

`StateStore` receives opaque names and byte strings. A node serializes its store
calls and treats writes and deletes as best effort, matching the official core.
Do not share one store between active nodes unless the store adds its own
synchronization, and do not call back into the node from a store method.

`NewFileStore` adapts `StateFS`, which embeds the standard `io/fs.FS` and adds
`WriteFile` and `Remove`. Directory creation, permissions, durability, and the
meaning of a write replacement remain embedding policy. Read failures with no
data are treated like missing official state objects; an unreadable cache must
not prevent the node from starting. Persisted peer files use the official v2
format and are cleaned after the official 30-day cache lifetime. After closing
a node that reported an identity collision, call `RotateIdentityState` on its
store before constructing the replacement node.

The top-level core never opens sockets or discovers host interfaces.
Embeddings may implement `WireSender` directly or use the optional `transport`
package, which invokes embedding-provided dialing and interface callbacks.

## Physical transport

The `transport` package owns one Node's primary and secondary IPv4/IPv6 UDP
sockets, opaque socket handles, connected UDP fallback, local direct paths,
secondary-port rotation, socket restoration, and TCP fail-forward state. A
Transport bounds endpoint-specific connected UDP sockets to 1024 entries and
evicts the least recently used entry when full. It is single-use: close it
before closing its Node, and create a new one when replacing the Node after an
identity collision or other restart. Shutdown uses `Transport.Close`, then
`Transport.Wait`, then `Node.Close`; `Wait` ensures that no physical receive
task can enter the node after its persistent state is handed to a replacement.

```go
wire, err := transport.New(transport.Config{
    Dialer:           dialer,
    Interfaces:       interfaces,
    SharedUDP:        true,
    PrimaryPort:      9993,
    SecondaryPort:    0,
    TCPFallbackMode:  transport.TCPFallbackAuto,
})
if err != nil {
    return err
}
node, err := zerotier.NewNode(zerotier.NodeConfig{
    Store:       store,
    Sender:      wire,
    DirectPaths: wire.DirectPaths,
})
if err != nil {
    return err
}
if err := wire.Start(ctx, node); err != nil {
    return err
}
defer func() {
    _ = wire.Close()
    wire.Wait()
    _ = node.Close()
}()
```

The injected `Dialer` retains control over proxies, interface binding, and
routing marks. The interface callback lets an embedding use its own cached or
platform-aware enumeration instead of forcing `net.Interfaces`. The package
does not implement per-interface Binder links or port mapping.

## TCP fallback

The `tcpfallback` package keeps relay protocol details out of an embedding's
socket adapter. Create a `Policy`, call `Reset` when a wire transport starts,
and serialize its methods with the embedding's transport state. For each
eligible global IPv4 send, `ObserveSend` reports whether fallback is required
and whether a missing connection should be dialed. A direct packet accepted by
`DirectEvidence` should be passed to `ObserveDirect` and, in automatic mode,
the active fallback session should be closed.

After the embedding dials its configured relay, `NewSession` sends the official
client hello. Install the session only if the node and transport generation are
still current, then run `Session.Run` with a handler that calls
`Node.ProcessWirePacket`. `Session.Send` performs record encoding and enforces
the official bounded write queue; `Session.Close` terminates both I/O paths.
Dialing, proxy selection, and logging intentionally remain embedding
responsibilities. `transport.Transport` supplies the lifecycle checks for its
managed sessions; custom `WireSender` implementations remain responsible for
their own stale-I/O protection.

## Layer-3 iplink

The `iplink` package adapts ZeroTier Ethernet frames to an IP packet device. It
applies managed routes, performs bounded IPv4 ARP and IPv6 Neighbor Discovery,
handles IGMP/MLD subscriptions, and retains ZeroTier RFC4193 and 6plane direct
address mapping.

```go
link, err := iplink.New(networkID, node)
if err != nil {
    return err
}
if err := link.ApplyNetworkConfig(config); err != nil {
    return err
}
go link.RunBackgroundTasks(ctx)

// L3 device to ZeroTier
err = link.WritePacket(ipPacket)

// ZeroTier to L3 device
ipPacket, err = link.HandleFrame(frame)
```

Run one `Link.RunBackgroundTasks` instance for the link lifetime. It retries
unresolved neighbors and failed multicast subscription changes every second,
drops pending packets after 10 seconds, and expires learned neighbors after 10
minutes. Sending ordinary multicast traffic does not subscribe the node;
dynamic subscriptions are learned exclusively from valid IGMP/MLD packets
written by the L3 device, while mandatory all-hosts and solicited-node groups
are maintained by the link itself. The link consumes validated inbound legacy
Reports to prevent another ZeroTier port from suppressing this port's own
IGMPv1/v2 or MLDv1 report; other control packets continue to the L3 device.
An observed General Query starts the RFC membership interval for previously
reported groups, and a subsequent local Report cancels that expiry. Without a
querier, dynamic state is retained rather than being removed by an unrelated
wall-clock timeout.
No method-level membership channel exists between the link and L3 stack.
An L3 stack that does not implement or enable IGMP/MLD remains supported; it
simply creates no dynamic multicast subscriptions.
`FrameSender` methods run inside a link packet or configuration transaction
and must not call back into the same link.

## ztprobe diagnostics

`cmd/ztprobe` is a manual interoperability and failure-localization tool. Its
network modes use `transport` with direct shared UDP, no secondary port, and no
TCP fallback, so a successful probe demonstrates planet reachability and the
same UDP hole-punching path used by an embedding. It persists its identity
under `-state` and supports private planets and repeatable `-orbit world,seed`
arguments.

```text
go run ./cmd/ztprobe config network.conf
go run ./cmd/ztprobe node -timeout 30s
go run ./cmd/ztprobe join -network 0123456789abcdef
go run ./cmd/ztprobe ping -network 0123456789abcdef -target 192.0.2.10
go run ./cmd/ztprobe ping -network 0123456789abcdef -target fd00::10 -size 1232 -json
go run ./cmd/ztprobe tcp -network 0123456789abcdef -target 192.0.2.10 -port 443
go run ./cmd/ztprobe tcp -network 0123456789abcdef -target fd00::10 -port 22 -mode connect -json
go run ./cmd/ztprobe http -network 0123456789abcdef -url http://service.example/health
go run ./cmd/ztprobe http -network 0123456789abcdef -url https://service.example/ -expect-status 200
go run ./cmd/ztprobe udp -network 0123456789abcdef -target 192.0.2.10 -port 9000 -send hello -expect world
go run ./cmd/ztprobe dns -network 0123456789abcdef -name example.com -type A
go run ./cmd/ztprobe mtu -network 0123456789abcdef -target 192.0.2.10
go run ./cmd/ztprobe fallback -network 0123456789abcdef
go run ./cmd/ztprobe watch -network 0123456789abcdef -target 192.0.2.10 -duration 10m
go run ./cmd/ztprobe restart -network 0123456789abcdef -cycles 3
```

`config` is offline. `node` verifies root connectivity. `join` verifies
controller status and prints assigned addresses, routes, DNS, MTU, auth URL,
and peer paths. `ping` sends IPv4 or IPv6 ICMP through `iplink`, covering
configuration, ARP/NDP, multicast delivery, and the final L3 path. `tcp` sends
IPv4 or IPv6 TCP SYN probes through the same path. A matching SYN-ACK reports
`open` and is immediately reset, a matching RST-ACK reports `closed`, and no
conclusive reply before `-timeout` reports `timeout`. `-count` bounds SYN
attempts. `tcp -mode connect` instead completes the third handshake ACK and an
orderly FIN close, including bounded FIN_WAIT_2 and TIME_WAIT handling, to
exercise the diagnostic TCP client without application data.

`http` performs one real HTTP/1.1 exchange over that TCP client. It supports
plain HTTP and standard-library TLS, controller-managed DNS or explicit
`-dns-server`/`-target` overrides, request methods, bodies and repeatable
headers, expected status/body checks, bounded response bodies, and direct-path
assertions. Redirects are reported rather than followed so one invocation has
a bounded and unambiguous destination. `-insecure` is available for deliberate
tests of private HTTPS endpoints with untrusted certificates.

`ping -count` sends that fixed number of requests. It validates the returned
payload as well as ICMP and IP checksums and reports loss, duplicates,
corruption, min/average/max RTT, and jitter. `-size` is the ICMP payload size.
`ping`, `tcp`, `udp`, `dns`, and `mtu` recognize matching IPv4 and IPv6 ICMP
errors instead of reducing unreachable, administratively prohibited, and
packet-too-big responses to timeouts. `-peer` attaches the expected ZeroTier
peer snapshot and `-require-direct` fails unless that peer establishes an
active physical path. RFC4193 and 6plane IPv6 targets infer their peer address.

`mtu` sends complete IP packets up to the managed network MTU and reports the
largest size whose echoed payload was intact. This exercises real ZeroTier
wire fragmentation and reassembly in both directions. `udp` exchanges an
application datagram with a service that replies; UDP silence alone cannot
distinguish an open port from filtering. `dns` uses Go's standard resolver
codec over a small raw-UDP adapter and defaults to the first controller-managed
DNS server. An explicit `-server IP[:PORT]` can be used when none is managed.
Single-label names use the managed DNS domain; other names are made absolute
so host OS search domains do not affect the result. The standard resolver can
retry truncated UDP responses over the diagnostic TCP client.

`fallback` disables UDP and requires the configured ZeroTier TCP relay before
reporting online or network-ready. This is deliberately separate from `node`,
whose success continues to prove shared UDP connectivity. `watch` emits an
event and probe timeline, accepts external authentication without exiting,
checks Ready/ConfigUpdate and Online/Offline ordering, tracks consecutive
failures and direct/relay samples, and verifies that the node ends online with
the network ready. JSON watch output is JSON Lines. `restart` repeatedly closes
and recreates the node and transport, requiring stable identity, cached network
configuration on later cycles, and a fresh controller response on every cycle.

Exit codes are stable for scripts: `0` success, `1` local/runtime error, `2`
usage error, `3` node or network not ready, `4` external authentication
required, and `5` L3 probe failure. `-json` emits machine-readable results.
TCP `closed` and `timeout` results use the existing L3 probe failure exit code.
Real-network probes are deliberately not part of the default unit test suite.
The TCP client is intentionally diagnostic-only: it implements active open,
MSS-sized stop-and-wait transmission, cumulative acknowledgements, bounded
overlap-aware out-of-order reassembly, retransmission, deadlines, orderly FIN
states, TIME_WAIT retransmission acknowledgement, and RST. It does not provide
listeners, a general connection pool, window scaling, SACK, or production
congestion control, and is not an embedding API.

## Deliberate exclusions

This project does not implement the embedded local controller, operating as a
root or relay, a local general-purpose Ethernet bridge, VLAN device plumbing,
port mapping, per-interface Binder behavior, or UDP TTL manipulation. A leaf
node can still learn and reach ordinary hosts behind an authorized remote
active bridge.

Enterprise SSO is not a target. The core parses and reports authentication
state so an embedding can expose a URL, but ZeroTier SSO v1 OIDC flows require
an official enterprise-capable client or a controller exemption.

The official baseline leaves its per-network AQM scheduler disabled
(`Network::qosEnabled()` returns false). PRIORITY rules and QoS path
measurements are wire-compatible, but packets are not placed into an inactive
official scheduler here.
