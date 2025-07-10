package evr

import "encoding/binary"

type LoginResult struct {
	XPID EvrId
}

func NewLoginResult(xpid EvrId) *LoginResult {
	return &LoginResult{
		XPID: xpid,
	}
}

func (r *LoginResult) Stream(s *EasyStream) error {
	var a uint32 = 1
	var status byte = 0x0b
	return RunErrorFunctions([]func() error{
		func() error { return s.StreamStruct(&r.XPID) },
		func() error { return s.StreamNumber(binary.LittleEndian, &a) }, // always 1
		func() error { return s.StreamByte(&status) },
		func() error { return s.Skip(3) }, // padding
	})
}
