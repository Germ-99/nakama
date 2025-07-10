package evr

import (
	"encoding/binary"

	"github.com/gofrs/uuid/v5"
)

// SNSLoginProfileResult
// LoginProfileResult is a message sent from server to client indicating a successful login profile request.
type LoginProfileResult struct {
	SessionID uuid.UUID
	XPID      EvrId
	Unk0      uint32
	UserData  LoginProfileRequest_UserData
	Profile   LoginProfileResult_UserProfile
}

func NewLoginProfileResult(sessionID uuid.UUID, serverProfile *ServerProfile) *LoginProfileResult {
	return &LoginProfileResult{
		SessionID: sessionID,
		XPID:      serverProfile.EvrID,
		Unk0:      0x00000000, // unknown value, always 0
		UserData: LoginProfileRequest_UserData{
			DisplayName:         serverProfile.DisplayName,
			UserID:              serverProfile.EvrID.String(),
			NPECompleted:        true,
			ArenaCombatUnlocked: true,
			PurchasedCombat:     true,
		},
		Profile: LoginProfileResult_UserProfile{
			Server: LoginProfileResult_Server{
				Loadout: LoginProfileResult_Loadout{
					Instances: []LoginProfileResult_LoadoutInstance{
						{
							Items: []LoginProfileResult_Item{
								{
									Item:     serverProfile.EquippedCosmetics.Instances.Unified.Slots.Emote,
									Itemslot: "emote1",
								},
								{
									Item:     serverProfile.EquippedCosmetics.Instances.Unified.Slots.Emote,
									Itemslot: "emote2",
								},
							},
							Name: "general",
						},
						{
							Items: []LoginProfileResult_Item{
								{Item: "pattern_default_head", Itemslot: "headpattern"},
								{Item: "decal_default", Itemslot: "headdecal1"},
								{Item: "decal_default", Itemslot: "headdecal2"},
								{Item: "tint_blue_a_default", Itemslot: "headtint"},
								{Item: "pattern_default_upper", Itemslot: "upperpattern"},
								{Item: "decal_default", Itemslot: "upperdecal1"},
								{Item: "decal_default", Itemslot: "upperdecal2"},
								{Item: "tint_blue_a_default", Itemslot: "uppertint"},
								{Item: "pattern_default_lower", Itemslot: "lowerpattern"},
								{Item: "decal_default", Itemslot: "lowerdecal"},
								{Item: "tint_blue_a_lower", Itemslot: "lowertint"},
							},
							Name: "blue",
						},
						{
							Items: []LoginProfileResult_Item{
								{Item: "pattern_default_head", Itemslot: "headpattern"},
								{Item: "decal_default", Itemslot: "headdecal1"},
								{Item: "decal_default", Itemslot: "headdecal2"},
								{Item: "tint_orange_a_default", Itemslot: "headtint"},
								{Item: "pattern_default_upper", Itemslot: "upperpattern"},
								{Item: "decal_default", Itemslot: "upperdecal1"},
								{Item: "decal_default", Itemslot: "upperdecal2"},
								{Item: "tint_orange_a_default", Itemslot: "uppertint"},
								{Item: "pattern_default_lower", Itemslot: "lowerpattern"},
								{Item: "decal_default", Itemslot: "lowerdecal"},
								{Item: "tint_orange_a_lower", Itemslot: "lowertint"},
							},
							Name: "orange",
						},
						{
							Items: []LoginProfileResult_Item{
								{Item: "pattern_default_head", Itemslot: "headpattern"},
								{Item: "decal_default", Itemslot: "headdecal1"},
								{Item: "decal_default", Itemslot: "headdecal2"},
								{Item: "tint_neutral_a_default", Itemslot: "headtint"},
								{Item: "pattern_default_upper", Itemslot: "upperpattern"},
								{Item: "decal_default", Itemslot: "upperdecal1"},
								{Item: "decal_default", Itemslot: "upperdecal2"},
								{Item: "tint_neutral_a_default", Itemslot: "uppertint"},
								{Item: "pattern_default_lower", Itemslot: "lowerpattern"},
								{Item: "decal_default", Itemslot: "lowerdecal"},
								{Item: "tint_neutral_a_lower", Itemslot: "lowertint"},
							},
							Name: "social",
						},
						{
							Items: []LoginProfileResult_Item{
								{Item: "pattern_default_head", Itemslot: "headpattern"},
								{Item: "decal_default", Itemslot: "headdecal1"},
								{Item: "decal_default", Itemslot: "headdecal2"},
								{Item: "tint_blue_a_default", Itemslot: "headtint"},
								{Item: "pattern_default_upper", Itemslot: "upperpattern"},
								{Item: "decal_default", Itemslot: "upperdecal1"},
								{Item: "decal_default", Itemslot: "upperdecal2"},
								{Item: "tint_blue_a_default", Itemslot: "uppertint"},
								{Item: "pattern_default_lower", Itemslot: "lowerpattern"},
								{Item: "decal_default", Itemslot: "lowerdecal"},
								{Item: "tint_blue_a_lower", Itemslot: "lowertint"},
							},
							Name: "blue_combat",
						},
						{
							Items: []LoginProfileResult_Item{
								{Item: "pattern_default_head", Itemslot: "headpattern"},
								{Item: "decal_default", Itemslot: "headdecal1"},
								{Item: "decal_default", Itemslot: "headdecal2"},
								{Item: "tint_orange_a_default", Itemslot: "headtint"},
								{Item: "pattern_default_upper", Itemslot: "upperpattern"},
								{Item: "decal_default", Itemslot: "upperdecal1"},
								{Item: "decal_default", Itemslot: "upperdecal2"},
								{Item: "tint_orange_a_default", Itemslot: "uppertint"},
								{Item: "pattern_default_lower", Itemslot: "lowerpattern"},
								{Item: "decal_default", Itemslot: "lowerdecal"},
								{Item: "tint_orange_a_lower", Itemslot: "lowertint"},
							},
							Name: "orange_combat",
						},
						{
							Items: []LoginProfileResult_Item{
								{Item: "pattern_default_head", Itemslot: "headpattern"},
								{Item: "decal_default", Itemslot: "headdecal1"},
								{Item: "decal_default", Itemslot: "headdecal2"},
								{Item: "tint_neutral_a_s10_default", Itemslot: "headtint"},
								{Item: "pattern_default_upper", Itemslot: "upperpattern"},
								{Item: "decal_default", Itemslot: "upperdecal1"},
								{Item: "decal_default", Itemslot: "upperdecal2"},
								{Item: "tint_neutral_a_s10_default", Itemslot: "uppertint"},
								{Item: "pattern_default_lower", Itemslot: "lowerpattern"},
								{Item: "decal_default", Itemslot: "lowerdecal"},
								{Item: "tint_neutral_a_s10_lower", Itemslot: "lowertint"},
							},
							Name: "social_combat",
						},
					},
					Number: int64(serverProfile.EquippedCosmetics.Number),
				},
			},
			ProfileStats: LoginProfileResult_ProfileStats{
				Level: LoginProfileResult_PlayerLevel{
					Cnt: 1,
					Op:  "set",
					Val: 1,
				},
			},
			ProfileStatsCombat: LoginProfileResult_ProfileStats{
				Level: LoginProfileResult_PlayerLevel{
					Cnt: 1,
					Op:  "set",
					Val: 1,
				},
			},
			Unlocks: []string{
				"emote_default",
				"emote_blink_smiley_a",
				"decal_default",
				"decal_sheldon_a",
				"pattern_default_head",
				"pattern_default_upper",
				"pattern_default_lower",
				"tint_neutral_a_default",
				"tint_blue_a_default",
				"tint_orange_a_default",
				"tint_neutral_a_lower",
				"tint_blue_a_lower",
				"tint_orange_a_lower",
				"tint_neutral_a_s10_default",
				"tint_neutral_a_s10_lower",
			},
			UnlocksCombat: []string{
				"emote_dizzy_eyes_a",
				"decal_combat_flamingo_a",
				"decal_combat_logo_a",
				"pattern_lightning_a_head",
				"pattern_lightning_a_upper",
				"pattern_lightning_a_lower",
			},

			XPlatformID:         serverProfile.EvrID.String(),
			Npecompleted:        true,
			ArenacombatUnlocked: true,
			Purchasedcombat:     true,
		},
	}
}

func (m *LoginProfileResult) Stream(s *EasyStream) error {
	result := byte(0x0b)
	return RunErrorFunctions([]func() error{
		func() error { return s.StreamGUID(&m.SessionID) },
		func() error { return s.StreamStruct(&m.XPID) },
		func() error { return s.StreamNumber(binary.LittleEndian, &m.Unk0) },
		func() error { return s.StreamByte(&result) },
		func() error { return s.Skip(3) }, // padding
		func() error { return s.StreamJson(&m.UserData, true, ZlibCompression) },
		func() error { return s.StreamJson(&m.Profile, false, ZlibCompression) },
	})
}

type LoginProfileRequest_UserData struct {
	DisplayName         string `json:"displayname"`
	UserID              string `json:"userid"`
	NPECompleted        bool   `json:"npecompleted"`
	ArenaCombatUnlocked bool   `json:"arenacombat_unlocked"`
	PurchasedCombat     bool   `json:"purchasedcombat"`
}

type LoginProfileResult_UserProfile struct {
	Server              LoginProfileResult_Server       `json:"server"`
	ProfileStats        LoginProfileResult_ProfileStats `json:"profile_stats"`
	ProfileStatsCombat  LoginProfileResult_ProfileStats `json:"profile_stats_combat"`
	Unlocks             []string                        `json:"unlocks"`
	UnlocksCombat       []string                        `json:"unlocks_combat"`
	XPlatformID         string                          `json:"xplatformid"`
	Npecompleted        bool                            `json:"npecompleted"`
	ArenacombatUnlocked bool                            `json:"arenacombat_unlocked"`
	Purchasedcombat     bool                            `json:"purchasedcombat"`
}

type LoginProfileResult_ProfileStats struct {
	Level LoginProfileResult_PlayerLevel `json:"Level"`
}

type LoginProfileResult_PlayerLevel struct {
	Cnt int64  `json:"cnt"`
	Op  string `json:"op"`
	Val int64  `json:"val"`
}

type LoginProfileResult_Server struct {
	Loadout LoginProfileResult_Loadout `json:"loadout"`
}

type LoginProfileResult_Loadout struct {
	Instances []LoginProfileResult_LoadoutInstance `json:"instances"`
	Number    int64                                `json:"number"`
}

type LoginProfileResult_LoadoutInstance struct {
	Items []LoginProfileResult_Item `json:"items"`
	Name  string                    `json:"name"`
}

type LoginProfileResult_Item struct {
	Item     string `json:"item"`
	Itemslot string `json:"itemslot"`
}
