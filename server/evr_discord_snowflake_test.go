package server

import (
	"sort"
	"strconv"
	"testing"

	"github.com/gofrs/uuid/v5"
)

func TestSnowflakeToUUID(t *testing.T) {
	t.Run("ValidSnowflake", func(t *testing.T) {
		snowflake := "695081603180789771"
		got := SnowflakeToUUID(snowflake)

		t.Log("Converted UUID:", got.String())
		if got == uuid.Nil {
			t.Errorf("Expected non-nil UUID for valid snowflake, got uuid.Nil")
		}
		if got.Version() != 8 {
			t.Errorf("Expected UUID version 8, got %d", got.Version())
		}
		if got.Variant() != uuid.VariantRFC4122 {
			t.Errorf("Expected UUID variant RFC4122, got %d", got.Variant())
		}
	})

	t.Run("ZeroSnowflake", func(t *testing.T) {
		snowflake := "0"
		u := SnowflakeToUUID(snowflake)
		if u == uuid.Nil {
			t.Errorf("Expected non-nil UUID for zero snowflake, got uuid.Nil")
		}
	})
}
func TestUUIDToSnowflake_ValidUUID(t *testing.T) {
	snowflake := "695081603180789771"
	u := SnowflakeToUUID(snowflake)
	id, err := UUIDToSnowflake(u)
	if err != nil {
		t.Errorf("Expected no error for valid UUID, got %v", err)
	}
	if strconv.FormatInt(id, 10) != snowflake {
		t.Errorf("Expected snowflake %s, got %d", snowflake, id)
	}
}

func TestUUIDToSnowflake_InvalidVersion(t *testing.T) {
	u := uuid.Must(uuid.NewV4())
	u.SetVariant(uuid.VariantRFC4122)
	id, err := UUIDToSnowflake(u)
	if err == nil {
		t.Errorf("Expected error for invalid UUID version, got nil")
	}
	if id != 0 {
		t.Errorf("Expected id 0 for invalid UUID version, got %d", id)
	}
}

func TestUUIDToSnowflake_InvalidVariant(t *testing.T) {
	snowflake := "695081603180789771"
	u := SnowflakeToUUID(snowflake)
	u.SetVariant(uuid.VariantNCS)
	id, err := UUIDToSnowflake(u)
	if err == nil {
		t.Errorf("Expected error for invalid UUID variant, got nil")
	}
	if id != 0 {
		t.Errorf("Expected id 0 for invalid UUID variant, got %d", id)
	}
}

func TestUUIDToSnowflake_NilUUID(t *testing.T) {
	id, err := UUIDToSnowflake(uuid.Nil)
	if err == nil {
		t.Errorf("Expected error for nil UUID, got nil")
	}
	if id != 0 {
		t.Errorf("Expected id 0 for nil UUID, got %d", id)
	}
}

func TestSnowflakeToUUID_InvalidSnowflake(t *testing.T) {
	invalidSnowflakes := []string{
		"",           // empty string
		"notanumber", // non-numeric
	}

	for _, s := range invalidSnowflakes {
		u := SnowflakeToUUID(s)
		if u != uuid.Nil {
			t.Errorf("Expected uuid.Nil for invalid snowflake '%s', got %v", s, u)
		}
	}
}

func TestUUIDToSnowflakeOrNil(t *testing.T) {
	snowflake := "695081603180789771"
	u := SnowflakeToUUID(snowflake)

	t.Run("WithUUID", func(t *testing.T) {
		result := UUIDToSnowflakeOrNil(u)
		if result != snowflake {
			t.Errorf("Expected %s, got %s", snowflake, result)
		}
	})

	t.Run("WithStringUUID", func(t *testing.T) {
		uStr := u.String()
		result := UUIDToSnowflakeOrNil(uStr)
		if result != snowflake {
			t.Errorf("Expected %s, got %s", snowflake, result)
		}
	})

	t.Run("InvalidUUIDString", func(t *testing.T) {
		invalidUUID := "not-a-uuid"
		result := UUIDToSnowflakeOrNil(invalidUUID)
		if result != "" {
			t.Errorf("Expected empty string for invalid UUID, got %s", result)
		}
	})

	t.Run("NilUUID", func(t *testing.T) {
		result := UUIDToSnowflakeOrNil(uuid.Nil)
		if result != "" {
			t.Errorf("Expected empty string for uuid.Nil, got %s", result)
		}
	})
}
func TestSnowflakeUUID_Sorting(t *testing.T) {
	snowflakes := []string{
		"695081603180789771",
		"695081603180789772",
		"695081603180789770",
		"695081603180789773",
	}
	// Convert snowflakes to UUIDs
	uuids := make([]uuid.UUID, len(snowflakes))
	for i, s := range snowflakes {
		uuids[i] = SnowflakeToUUID(s)
	}

	// Shuffle the UUIDs
	shuffled := make([]uuid.UUID, len(uuids))
	copy(shuffled, uuids)
	for i := range shuffled {
		j := i + 1
		if j < len(shuffled) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		}
	}

	// Sort by UUID bytes
	sorted := make([]uuid.UUID, len(shuffled))
	copy(sorted, shuffled)
	// Sort using bytes.Compare
	// This works because the encoding preserves snowflake order
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].String() < sorted[j].String()
	})

	// Convert sorted UUIDs back to snowflakes
	sortedSnowflakes := make([]string, len(sorted))
	for i, u := range sorted {
		id, err := UUIDToSnowflake(u)
		if err != nil {
			t.Fatalf("Failed to convert UUID to snowflake: %v", err)
		}
		sortedSnowflakes[i] = strconv.FormatInt(id, 10)
	}

	// Check that the sorted snowflakes are in ascending order
	for i := 1; i < len(sortedSnowflakes); i++ {
		if sortedSnowflakes[i-1] > sortedSnowflakes[i] {
			t.Errorf("Snowflakes not sorted: %v", sortedSnowflakes)
			break
		}
	}
}
func TestSnowflakeToUUID_Fuzzy(t *testing.T) {
	// Fuzzy test: generate random snowflake IDs and check round-trip conversion
	for i := range 1000 {
		// Discord snowflakes are 64-bit integers, but typically fit in 53 bits for JS compatibility
		id := int64(1<<52) + int64(i)*1234567 // start at a large value, step to avoid collisions
		snowflake := strconv.FormatInt(id, 10)
		u := SnowflakeToUUID(snowflake)
		if u == uuid.Nil {
			t.Errorf("Fuzzy: Expected non-nil UUID for snowflake %s", snowflake)
			continue
		}
		roundTrip, err := UUIDToSnowflake(u)
		if err != nil {
			t.Errorf("Fuzzy: Error converting UUID back to snowflake: %v", err)
			continue
		}
		if roundTrip != id {
			t.Errorf("Fuzzy: Round-trip mismatch: snowflake %s, got %d", snowflake, roundTrip)
		}
	}
}

func TestUUIDToSnowflake_Fuzzy(t *testing.T) {
	// Fuzzy test: generate random snowflake IDs, convert to UUID, then back
	for i := range 1000 {
		id := int64(1<<52) + int64(i)*9876543
		snowflake := strconv.FormatInt(id, 10)
		u := SnowflakeToUUID(snowflake)
		if u == uuid.Nil {
			t.Errorf("Fuzzy: Expected non-nil UUID for snowflake %s", snowflake)
			continue
		}
		// Mutate UUID slightly and check error handling
		badUUID := u
		badUUID.SetVersion(4)
		_, err := UUIDToSnowflake(badUUID)
		if err == nil {
			t.Errorf("Fuzzy: Expected error for bad UUID version, got nil")
		}
		// Test valid round-trip
		roundTrip, err := UUIDToSnowflake(u)
		if err != nil {
			t.Errorf("Fuzzy: Error converting UUID back to snowflake: %v", err)
			continue
		}
		if roundTrip != id {
			t.Errorf("Fuzzy: Round-trip mismatch: snowflake %s, got %d", snowflake, roundTrip)
		}
	}
}
