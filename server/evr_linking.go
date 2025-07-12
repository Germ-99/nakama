package server

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
)

const (
	LinkTicketCodeLength          = 4
	LinkTicketLifetimeDays        = 30
	LinkTicketCodeValidCharacters = "ACDEFGHIJKLMPRSTUXZ"

	StorageCollectionAuthorization = "Authorization"
)

var (
	ErrLinkNotFound = fmt.Errorf("link code not found")
)

// LinkTicket represents a link ticket used for linking a headset to a user account.
type LinkTicket struct {
	XPID         evr.EvrId `json:"xp_id"`         // the xplatform ID used by the client/server
	ClientIP     string    `json:"client_ip"`     // the client IP address that generated this link ticket
	LoginPayload string    `json:"login_payload"` // the login request payload that generated this link ticket
	CreatedAt    time.Time `json:"created_at"`    // the time the link ticket was created
}

// LinkTickets is a storage structure for managing link tickets.
type LinkTickets struct {
	Tickets map[string]*LinkTicket `json:"tickets"`
	version string
}

// Prune removes expired link tickets from the storage.
func (s *LinkTickets) pruneOldTickets() {
	for code, ticket := range s.Tickets {
		if time.Since(ticket.CreatedAt) > LinkTicketLifetimeDays*24*time.Hour {
			delete(s.Tickets, code)
		}
	}
}

// GetByXPID retrieves a link ticket by its xplatform ID.
func (s *LinkTickets) GetCodeByXPID(xpID evr.EvrId) string {
	for code, ticket := range s.Tickets {
		if ticket.XPID == xpID {
			return code
		}
	}
	return ""
}

// GenerateTicket creates a new link ticket with a unique code.
func (s *LinkTickets) GenerateCode(xpid evr.EvrId, clientIP, payload string) string {
	// Ensure the tickets map is initialized
	if s.Tickets == nil {
		s.Tickets = make(map[string]*LinkTicket)
	}

	// Prune old tickets before generating a new one
	s.pruneOldTickets()

	// Create a new local random generator with a known seed value
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a unique link code
	var code string
	for {

		// Randomly select an index from the array and generate the code
		var b []byte
		for range LinkTicketCodeLength {
			index := rng.Intn(len(LinkTicketCodeValidCharacters))
			b = append(b, LinkTicketCodeValidCharacters[index])
		}
		code = string(b)
		if _, ok := s.Tickets[code]; !ok {
			break
		}
	}

	// Create a new link ticket
	s.Tickets[code] = &LinkTicket{
		XPID:         xpid,
		ClientIP:     clientIP,
		LoginPayload: payload,
		CreatedAt:    time.Now(),
	}

	return code
}

// LoadAndDelete retrieves and removes a link ticket by its code.
func (s *LinkTickets) LoadAndDelete(linkCode string) (xpid evr.EvrId, clientIP, payload string, found bool) {
	if s.Tickets == nil {
		return
	}
	ticket, ok := s.Tickets[linkCode]
	if !ok {
		return
	}
	delete(s.Tickets, linkCode)
	return ticket.XPID, ticket.ClientIP, ticket.LoginPayload, true
}

// ExchangeLinkCode exchanges a link code for a link ticket.
func ExchangeLinkCode(ctx context.Context, nk runtime.NakamaModule, logger runtime.Logger, linkCode string) (xpid evr.EvrId, clientIP, payload string, err error) {
	var found bool

	// Normalize the link code to uppercase.
	linkTickets := &LinkTickets{}
	if err := StorageRead(ctx, nk, SystemUserID, linkTickets, true); err != nil {
		return evr.EvrId{}, "", "", fmt.Errorf("failed to read link tickets from storage: %w", err)
	}

	xpid, clientIP, payload, found = linkTickets.LoadAndDelete(linkCode)
	if !found {
		return evr.EvrId{}, "", "", ErrLinkNotFound
	}

	if err := StorageWrite(ctx, nk, SystemUserID, linkTickets); err != nil {
		return evr.EvrId{}, "", "", fmt.Errorf("failed to write link tickets to storage: %w", err)
	}

	return xpid, clientIP, payload, nil
}

func (s LinkTickets) StorageMeta() StorageMeta {
	return StorageMeta{
		Collection:      StorageCollectionAuthorization,
		Key:             "linkTickets",
		PermissionRead:  0,
		PermissionWrite: 0,
		Version:         s.version,
	}
}

func (LinkTickets) StorageIndexes() []StorageIndexMeta { return nil }

func (s *LinkTickets) SetStorageVersion(userID string, version string) { s.version = version }
