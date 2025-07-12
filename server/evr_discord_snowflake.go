package server

import (
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

	// Set the version to 8 (1000) in the most significant 4 bits of octet 6.
	u[6] = 0x80

	// Set the variant to RFC 4122 (10xx xxxx) in the most significant 2 bits of octet 8.
	u[8] = 0x20

	// Encode the first 4 bytes with the most significant 48 bits of the snowflake ID.
	u[0] = byte((id >> 40) & 0xFF)
	u[1] = byte((id >> 32) & 0xFF)
	u[2] = byte((id >> 24) & 0xFF)
	u[3] = byte((id >> 16) & 0xFF)
	u[4] = byte((id >> 8) & 0xFF)
	u[5] = byte(id & 0xFF)

	// Starting at byte 9, encode the least significant 24 bits of the snowflake ID.
	u[9] = byte((id >> 16) & 0xFF)
	u[10] = byte((id >> 8) & 0xFF)
	u[11] = byte(id & 0xFF)

	return u

}

func UUIDToSnowflake[T uuid.UUID | string](snowUUID T) string {
	u := uuid.UUID{}
	if v, ok := any(snowUUID).(string); ok {
		u = uuid.FromStringOrNil(v)
	}

	if u[6]>>4 != 8 {
		return ""
	}
	if u[8]>>6 != 0x02 {
		return ""
	}

	// Extract the snowflake ID from the UUID.
	id := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 |
		int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5]) |
		int64(u[9])<<16 | int64(u[10])<<8 | int64(u[11])

	return strconv.FormatInt(id, 10)
}
