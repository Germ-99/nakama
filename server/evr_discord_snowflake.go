package server

import (
	"fmt"
	"strconv"

	"github.com/gofrs/uuid/v5"
)

// SnowflakeToUUID returns a UUID representation of a snowflake ID string.
func SnowflakeToUUID(s string) uuid.UUID {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return uuid.Nil
	}
	u := uuid.UUID{}
	u.SetVersion(8) // Set UUID version to 8 (custom)
	u.SetVariant(uuid.VariantRFC4122)
	// Encode the first 4 bytes with the most significant 48 bits of the snowflake ID.
	fmt.Printf("%08b\n", u)
	u[0] = byte((id >> 40) & 0xFF)
	u[1] = byte((id >> 32) & 0xFF)
	u[2] = byte((id >> 24) & 0xFF)
	u[3] = byte((id >> 16) & 0xFF)
	u[4] = byte((id >> 8) & 0xFF)
	u[5] = byte(id & 0xFF)
	fmt.Printf("%08b\n", u)
	// Starting at byte 9, encode the least significant 24 bits of the snowflake ID.
	u[9] = byte((id >> 16) & 0xFF)
	u[10] = byte((id >> 8) & 0xFF)
	u[11] = byte(id & 0xFF)
	fmt.Printf("%08b\n", u)
	return u
}

// UUIDToSnowflake converts a UUID back to a snowflake ID.
func UUIDToSnowflake(u uuid.UUID) (int64, error) {
	if u.Version() != 8 {
		return 0, fmt.Errorf("invalid UUID version: %d", u.Version())
	}
	if u.Variant() != uuid.VariantRFC4122 {
		return 0, fmt.Errorf("invalid UUID variant: %d", u.Variant())
	}
	id := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 |
		int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5]) |
		int64(u[9])<<16 | int64(u[10])<<8 | int64(u[11])
	return id, nil
}

func UUIDToSnowflakeOrNil[T uuid.UUID | string](snowUUID T) string {
	u := uuid.UUID{}
	if v, ok := any(snowUUID).(string); ok {
		u = uuid.FromStringOrNil(v)
	}
	id, err := UUIDToSnowflake(u)
	if err != nil {
		return ""
	}
	return strconv.FormatInt(id, 10)
}
