package server

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/heroiclabs/nakama/v3/server/evr"
	"go.uber.org/zap"
)

var (
	DevicePrefixXPID       = "xp_id:"
	DevicePrefixVRMLUserID = "vrml_user_id:"
	DevicePrefixTokenID    = "token_id:"
	DeviceXPID             = func(xpid evr.EvrId) string { return DevicePrefixXPID + xpid.String() }
	DeviceVRMLUserID       = func(vrmlUserID string) string { return DevicePrefixVRMLUserID + vrmlUserID }
	DeviceTokenID          = func(vrmlUserID string) string { return DevicePrefixTokenID + vrmlUserID }
)

// LinkXPID links an XPID to a user account in Nakama.
func LinkXPID(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, xpID evr.EvrId) error {
	if err := LinkDevice(ctx, logger, db, userID, DeviceXPID(xpID)); err != nil {
		return fmt.Errorf("failed to link XPID: %w", err)
	}
	return nil
}

func LinkVRMLUser(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, vrmlUserID string) error {
	if err := LinkDevice(ctx, logger, db, userID, DeviceVRMLUserID(vrmlUserID)); err != nil {
		return fmt.Errorf("failed to link VRML account: %w", err)
	}
	return nil
}

func LinkClientTokenID(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, tokenID string) error {
	if err := LinkDevice(ctx, logger, db, userID, DeviceTokenID(tokenID)); err != nil {
		return fmt.Errorf("failed to link client token id: %w", err)
	}

	return nil
}

func UnlinkXPID(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, xpID evr.EvrId) error {
	if err := UnlinkDevice(ctx, logger, db, userID, DeviceXPID(xpID)); err != nil {
		return fmt.Errorf("failed to unlink XPID: %w", err)
	}
	return nil
}

func UnlinkVRMLUser(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, vrmlUserID string) error {
	if err := UnlinkDevice(ctx, logger, db, userID, DeviceVRMLUserID(vrmlUserID)); err != nil {
		return fmt.Errorf("failed to unlink VRML account: %w", err)
	}
	return nil
}

func UnlinkClientTokenID(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, tokenID string) error {
	if err := UnlinkDevice(ctx, logger, db, userID, DeviceTokenID(tokenID)); err != nil {
		return fmt.Errorf("failed to unlink client token id: %w", err)
	}
	return nil
}
