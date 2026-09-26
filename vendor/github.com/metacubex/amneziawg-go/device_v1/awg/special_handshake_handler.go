package awg

import (
	"errors"
	"sync/atomic"
	"time"
)

// TODO: atomic?/ and better way to use this
var PacketCounter atomic.Uint64

// TODO
var WaitResponse = struct {
	Channel    chan struct{}
	ShouldWait atomic.Bool
}{
	make(chan struct{}, 1),
	atomic.Bool{},
}

type SpecialHandshakeHandler struct {
	isFirstDone    bool
	SpecialJunk    TagJunkPacketGenerators
	ControlledJunk TagJunkPacketGenerators

	nextItime time.Time
	ITimeout  time.Duration // seconds

	IsSet bool
}

func (handler *SpecialHandshakeHandler) Validate() error {
	var errs []error
	if err := handler.SpecialJunk.Validate(); err != nil {
		errs = append(errs, err)
	}
	if err := handler.ControlledJunk.Validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (handler *SpecialHandshakeHandler) GenerateSpecialJunk(protocol *Protocol) [][]byte {
	if !handler.SpecialJunk.IsDefined() {
		return nil
	}

	// TODO: create tests
	if !handler.isFirstDone {
		handler.isFirstDone = true
	} else if !handler.isTimeToSendSpecial() {
		return nil
	}

	rv := handler.SpecialJunk.GeneratePackets(protocol)
	handler.nextItime = time.Now().Add(handler.ITimeout)

	return rv
}

func (handler *SpecialHandshakeHandler) isTimeToSendSpecial() bool {
	return time.Now().After(handler.nextItime)
}

func (handler *SpecialHandshakeHandler) GenerateControlledJunk(protocol *Protocol) [][]byte {
	if !handler.ControlledJunk.IsDefined() {
		return nil
	}

	return handler.ControlledJunk.GeneratePackets(protocol)
}
