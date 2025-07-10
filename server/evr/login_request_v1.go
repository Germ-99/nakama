package evr

import (
	"encoding/binary"
	"fmt"

	"github.com/gofrs/uuid/v5"
)

// LoginRequestV1 represents a message from client to server requesting for a user sign-in.
type LoginRequestV1 struct {
	PreviousSessionID uuid.UUID // This is the old session id, if it had one.
	XPID              EvrId
	Payload           LoginRequestV1_Profile
}

func (lr LoginRequestV1) String() string {
	return fmt.Sprintf("%T(Session=%s, XPID=%s, HMDSerialNumber=%s)", lr, lr.PreviousSessionID, lr.XPID, lr.Payload.HMDSerialNumber)
}

func (m *LoginRequestV1) Stream(s *EasyStream) error {
	var randomNumber int32 = 0 // Placeholder for a random int64, if needed in the future.
	return RunErrorFunctions([]func() error{
		func() error { return s.StreamGUID(&m.PreviousSessionID) },
		func() error { return s.StreamStruct(&m.XPID) },
		func() error { return s.StreamNumber(binary.LittleEndian, &randomNumber) }, // Placeholder for a random int64
		func() error { return s.StreamJson(&m.Payload, true, NoCompression) },
	})
}

func (m *LoginRequestV1) GetEvrID() EvrId {
	return m.XPID
}

type LoginRequestV1_Profile struct {
	AppID           int64  `json:"appid"`
	AccountID       int64  `json:"accountid"`
	AccessToken     string `json:"access_token"`
	LobbyVersion    int64  `json:"lobbyversion"`
	Nonce           string `json:"nonce"`
	PublisherLock   string `json:"publisher_lock"`
	HMDSerialNumber string `json:"hmdserialnumber"`
	ExtraField1     string `json:"extra_field_1"`
	ExtraField2     int64  `json:"extra_field_2"`
}
