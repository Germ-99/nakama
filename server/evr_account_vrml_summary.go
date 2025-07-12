package server

import (
	"errors"

	"github.com/echotools/vrmlgo/v5"
)

const (
	StorageCollectionVRML  = "VRML"
	StorageKeyVRMLSummary  = "summary"
	StorageIndexVRMLUserID = "Index_VRMLUserID"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
)

type VRMLPlayerSummary struct {
	User                      *vrmlgo.User                    `json:"user"`
	Player                    *vrmlgo.Player                  `json:"player"`
	Teams                     map[string]*vrmlgo.Team         `json:"teams"`        // map[teamID]team
	MatchCountsBySeasonByTeam map[VRMLSeasonID]map[string]int `json:"match_counts"` // map[seasonID]map[teamID]matchCount
}

func (VRMLPlayerSummary) StorageMeta() StorageMeta {
	return StorageMeta{
		Collection: StorageCollectionVRML,
		Key:        StorageKeyVRMLSummary,
	}
}

func (VRMLPlayerSummary) StorageIndexes() []StorageIndexMeta {

	// Register the storage index
	return []StorageIndexMeta{
		{
			Name:       StorageIndexVRMLUserID,
			Collection: StorageCollectionVRML,
			Key:        StorageKeyVRMLSummary,
			Fields:     []string{"userID"},
			MaxEntries: 1000000,
			IndexOnly:  true,
		},
	}
}

func (s *VRMLPlayerSummary) Entitlements() []*VRMLEntitlement {

	matchCountBySeason := make(map[VRMLSeasonID]int)
	for sID, teamID := range s.MatchCountsBySeasonByTeam {
		for _, c := range teamID {
			matchCountBySeason[sID] += c
		}
	}

	// Validate match counts
	entitlements := make([]*VRMLEntitlement, 0)

	for seasonID, matchCount := range matchCountBySeason {

		switch seasonID {

		// Pre-season and Season 1 have different requirements
		case VRMLPreSeason, VRMLSeason1:
			if matchCount > 0 {
				entitlements = append(entitlements, &VRMLEntitlement{
					SeasonID: seasonID,
				})
			}

		default:
			if matchCount >= 10 {
				entitlements = append(entitlements, &VRMLEntitlement{
					SeasonID: seasonID,
				})
			}
		}
	}

	return entitlements
}
