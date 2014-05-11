// Copyright 2014 Mikio Hara. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64 arm

package tcp

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

func (opt *opt) info() (*Info, error) {
	fd, err := opt.sysfd()
	if err != nil {
		return nil, err
	}
	var v syscall.TCPInfo
	l := sysSockoptLen(syscall.SizeofTCPInfo)
	if err := getsockopt(fd, syscall.IPPROTO_TCP, syscall.TCP_INFO, unsafe.Pointer(&v), &l); err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}
	return parseTCPInfo(&v), nil
}

var sysStates = [12]State{Unknown, Established, SynSent, SynReceived, FinWait1, FinWait2, TimeWait, Closed, CloseWait, LastAck, Listen, Closing}

func parseTCPInfo(sti *syscall.TCPInfo) *Info {
	ti := &Info{State: sysStates[sti.State]}
	if sti.Options&sysTCPIOptWscale != 0 {
		ti.Options = append(ti.Options, WindowScale(sti.Pad_cgo_0[0]>>4))
		ti.PeerOptions = append(ti.PeerOptions, WindowScale(sti.Pad_cgo_0[0]&0x0f))
	}
	if sti.Options&sysTCPIOptTimestamps != 0 {
		ti.Options = append(ti.Options, Timestamps(true))
		ti.PeerOptions = append(ti.PeerOptions, Timestamps(true))
	}
	ti.SenderMSS = MaxSegSize(sti.Snd_mss)
	ti.ReceiverMSS = MaxSegSize(sti.Rcv_mss)
	ti.LastDataSent = time.Duration(sti.Last_data_sent) * time.Millisecond
	ti.LastDataReceived = time.Duration(sti.Last_data_recv) * time.Millisecond
	ti.LastAckReceived = time.Duration(sti.Last_ack_recv) * time.Millisecond
	ti.CC = &CongestionControl{
		RTO:                 time.Duration(sti.Rto) * time.Microsecond,
		ATO:                 time.Duration(sti.Ato) * time.Microsecond,
		RTT:                 time.Duration(sti.Rtt) * time.Microsecond,
		RTTStdDev:           time.Duration(sti.Rttvar) * time.Microsecond,
		SenderSSThreshold:   uint(sti.Snd_ssthresh),
		ReceiverSSThreshold: uint(sti.Rcv_ssthresh),
		SenderWindow:        uint(sti.Snd_cwnd),
	}
	ti.SysInfo = &SysInfo{
		AdvertisedMSS:  MaxSegSize(sti.Advmss),
		ReceiverWindow: uint(sti.Rcv_space),
		CAState:        CAState(sti.Ca_state),
		UnackSegs:      uint(sti.Unacked),
		SackSegs:       uint(sti.Sacked),
		LostSegs:       uint(sti.Lost),
		RetransSegs:    uint(sti.Retrans),
		ForwardAckSegs: uint(sti.Fackets),
	}
	return ti
}