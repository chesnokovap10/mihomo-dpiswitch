#!/bin/bash

set -ev

go test github.com/metacubex/gopacket
go test github.com/metacubex/gopacket/layers
go test github.com/metacubex/gopacket/tcpassembly
go test github.com/metacubex/gopacket/reassembly
go test github.com/metacubex/gopacket/pcapgo
go test github.com/metacubex/gopacket/pcap
sudo $(which go) test github.com/metacubex/gopacket/routing
