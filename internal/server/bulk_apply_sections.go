package server

// RunDetailDefinition describes a run that has configurable details
type RunDetailDefinition struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	HasDetails bool   `json:"hasDetails"`
}

// BulkApplyRunDetails defines runs with configurable detail options
// NOTE: Section definitions are now auto-discovered from CharacterCfg struct tags
// via config.DiscoverBulkApplySections() - see internal/config/bulkapply.go
// This list should match all case statements in applyRunDetails
var BulkApplyRunDetails = []RunDetailDefinition{
	{ID: "andariel", Label: "Andariel", HasDetails: true},
	{ID: "countess", Label: "Countess", HasDetails: true},
	{ID: "duriel", Label: "Duriel", HasDetails: true},
	{ID: "pit", Label: "Pit", HasDetails: true},
	{ID: "cows", Label: "Cows", HasDetails: true},
	{ID: "pindleskin", Label: "Pindleskin", HasDetails: true},
	{ID: "stony_tomb", Label: "Stony Tomb", HasDetails: true},
	{ID: "mausoleum", Label: "Mausoleum", HasDetails: true},
	{ID: "ancient_tunnels", Label: "Ancient Tunnels", HasDetails: true},
	{ID: "drifter_cavern", Label: "Drifter Cavern", HasDetails: true},
	{ID: "spider_cavern", Label: "Spider Cavern", HasDetails: true},
	{ID: "arachnid_lair", Label: "Arachnid Lair", HasDetails: true},
	{ID: "mephisto", Label: "Mephisto", HasDetails: true},
	{ID: "tristram", Label: "Tristram", HasDetails: true},
	{ID: "nihlathak", Label: "Nihlathak", HasDetails: true},
	{ID: "summoner", Label: "Summoner", HasDetails: true},
	{ID: "baal", Label: "Baal", HasDetails: true},
	{ID: "eldritch", Label: "Eldritch", HasDetails: true},
	{ID: "lower_kurast_chest", Label: "Lower Kurast Chest", HasDetails: true},
	{ID: "diablo", Label: "Diablo", HasDetails: true},
	{ID: "leveling", Label: "Leveling", HasDetails: true},
	{ID: "leveling_sequence", Label: "Leveling Sequence", HasDetails: true},
	{ID: "quests", Label: "Quests", HasDetails: true},
	{ID: "terror_zone", Label: "Terror Zone", HasDetails: true},
	{ID: "utility", Label: "Utility", HasDetails: true},
	{ID: "shopping", Label: "Shopping", HasDetails: false}, // Handled by Shopping section
}

// GetAllRunDetailIDs returns all run IDs that have configurable details
func GetAllRunDetailIDs() []string {
	result := make([]string, 0, len(BulkApplyRunDetails))
	for _, rd := range BulkApplyRunDetails {
		if rd.HasDetails {
			result = append(result, rd.ID)
		}
	}
	return result
}
