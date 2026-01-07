package action

import (
	"fmt"
	"strings"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/stat"

	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/pickit"
)

// IsDropProtected determines which items must NOT be dropped
func IsDropProtected(i data.Item) bool {
	ctx := context.Get()
	selected := false
	DropperOnly := false
	filtersEnabled := false
	usePickit := false

	if ctx != nil && ctx.Context != nil {
		if ctx.Context.Drop != nil {
			filtersEnabled = ctx.Context.Drop.DropFiltersEnabled()
			usePickit = ctx.Context.Drop.UsePickitEnabled()
			if filtersEnabled && !usePickit {
				selected = ctx.Context.Drop.ShouldDropperItem(string(i.Name), i.Quality, i.Type().Code, i.IsRuneword)
				DropperOnly = ctx.Context.Drop.DropperOnlySelected()
			}
		}
	}

	// Always keep the cube so the bot can continue farming afterward.
	if i.Name == "HoradricCube" {
		return true
	}

	// Protect runeword reroll targets (and their temporary bases) from Drop.
	if shouldProtectRunewordReroll(ctx, i) {
		return true
	}

	// UsePickit mode: drop items that MATCH pickit rules (good items)
	if usePickit {
		// Always protect essential consumables - bot needs these to function
		if isEssentialConsumable(i) {
			return true
		}

		// Keep recipe materials first
		if shouldKeepRecipeItem(i) {
			return true
		}

		// Check if item matches pickit rules
		if MatchesPickitRules(i) {
			// Item matches pickit - check category toggles to see if we can drop it
			// Default to protecting gems/jewels if we can't check the toggles
			dropMgr := ctx.Context.Drop
			if dropMgr == nil {
				// Can't check toggles, protect the item to be safe
				return true
			}

			// Gems: protected by default, drop only if DropGems is enabled
			if isGemItem(i) && !dropMgr.DropGemsEnabled() {
				return true // protected
			}
			// Jewels: protected by default, drop only if DropJewels is enabled
			if i.Name == "Jewel" && !dropMgr.DropJewelsEnabled() {
				return true // protected
			}
			// Charms: no special protection - skillers match pickit and get dropped,
			// reroll candidates don't match pickit and are already protected below

			// Item matches pickit and isn't in a protected category - drop it
			return false
		}
		// Item doesn't match pickit - protect it (don't drop)
		return true
	}

	if selected {
		if ctx != nil && ctx.Context != nil && ctx.Context.Drop != nil && !ctx.Context.Drop.HasRemainingDropQuota(string(i.Name)) {
			return true
		}
		return false
	}

	// Keep recipe materials configured in cube settings.
	if shouldKeepRecipeItem(i) {
		return true
	}

	if i.Name == "GrandCharm" && ctx != nil && HasGrandCharmRerollCandidate(ctx) {
		return true
	}

	if !filtersEnabled {
		return false
	}

	if DropperOnly {
		return true
	}

	// Everything else should be dropped for Drop to ensure the stash empties fully.
	return false
}

func shouldProtectRunewordReroll(ctx *context.Status, itm data.Item) bool {
	if ctx == nil || ctx.CharacterCfg == nil {
		return false
	}
	if !ctx.CharacterCfg.Game.RunewordMaker.Enabled {
		return false
	}
	if _, isLevelingChar := ctx.Char.(context.LevelingCharacter); isLevelingChar {
		return false
	}
	if len(ctx.CharacterCfg.Game.RunewordRerollRules) == 0 {
		return false
	}

	if shouldProtectRunewordRerollItem(ctx, itm) {
		return true
	}
	if shouldProtectRunewordRerollBase(ctx, itm) {
		return true
	}

	return false
}

func shouldProtectRunewordRerollItem(ctx *context.Status, itm data.Item) bool {
	if !itm.IsRuneword {
		return false
	}

	rules := ctx.CharacterCfg.Game.RunewordRerollRules[string(itm.RunewordName)]
	if len(rules) == 0 {
		return false
	}

	applicableRuleFound, meetsAnyRule, _ := evaluateRunewordRules(ctx, itm, rules, string(itm.RunewordName))
	if !applicableRuleFound || meetsAnyRule {
		return false
	}

	return true
}

func shouldProtectRunewordRerollBase(ctx *context.Status, itm data.Item) bool {
	if itm.IsRuneword || itm.HasSocketedItems() {
		return false
	}

	sockets, found := itm.FindStat(stat.NumSockets, 0)
	if !found {
		return false
	}

	for _, recipe := range Runewords {
		rules := ctx.CharacterCfg.Game.RunewordRerollRules[string(recipe.Name)]
		if len(rules) == 0 {
			continue
		}

		if sockets.Value != len(recipe.Runes) {
			continue
		}

		if !matchesRunewordBaseType(itm, recipe) {
			continue
		}

		for _, rule := range rules {
			if baseMatchesRerollRule(itm, rule) {
				return true
			}
		}
	}

	return false
}

func matchesRunewordBaseType(itm data.Item, recipe Runeword) bool {
	itemType := itm.Type().Code
	for _, baseType := range recipe.BaseItemTypes {
		if itemType == baseType {
			return true
		}
	}
	return false
}

func baseMatchesRerollRule(itm data.Item, rule config.RunewordRerollRule) bool {
	desc := itm.Desc()
	baseCode := pickit.ToNIPName(desc.Name)

	ethMode := strings.ToLower(strings.TrimSpace(rule.EthMode))
	switch ethMode {
	case "eth":
		if !itm.Ethereal {
			return false
		}
	case "noneth":
		if itm.Ethereal {
			return false
		}
	}

	qualityMode := strings.ToLower(strings.TrimSpace(rule.QualityMode))
	switch qualityMode {
	case "normal":
		if itm.Quality != item.QualityNormal {
			return false
		}
	case "superior":
		if itm.Quality != item.QualitySuperior {
			return false
		}
	}

	baseNameExplicitMatch := false
	if rule.BaseName != "" {
		for _, part := range strings.Split(rule.BaseName, ",") {
			if strings.TrimSpace(part) == baseCode {
				baseNameExplicitMatch = true
				break
			}
		}
		if !baseNameExplicitMatch {
			return false
		}
	}

	if !baseNameExplicitMatch {
		if rule.BaseType != "" && desc.Type != rule.BaseType {
			return false
		}

		if rule.BaseTier != "" {
			switch strings.ToLower(strings.TrimSpace(rule.BaseTier)) {
			case "normal":
				if desc.Tier() != item.TierNormal {
					return false
				}
			case "exceptional":
				if desc.Tier() != item.TierExceptional {
					return false
				}
			case "elite":
				if desc.Tier() != item.TierElite {
					return false
				}
			}
		}
	}

	return true
}

func RunDropCleanup() error {
	ctx := context.Get()

	ctx.RefreshGameData()

	if !ctx.Data.PlayerUnit.Area.IsTown() {
		if err := ReturnTown(); err != nil {
			return fmt.Errorf("failed to return to town for Drop cleanup: %w", err)
		}
		// Update town/NPC data after the town portal sequence.
		ctx.RefreshGameData()
	}
	RecoverCorpse()

	IdentifyAll(false)
	ctx.PauseIfNotPriority()
	Stash(false)
	ctx.PauseIfNotPriority()
	VendorRefill(VendorRefillOpts{SellJunk: true})
	ctx.PauseIfNotPriority() // Check after VendorRefill
	Stash(false)
	ctx.PauseIfNotPriority() // Check after Stash

	ctx.RefreshGameData()
	if ctx.Data.OpenMenus.IsMenuOpen() {
		step.CloseAllMenus()
	}
	return nil
}

// HasGrandCharmRerollCandidate indicates whether a reroll-able GrandCharm + perfect gems exist in stash.
func HasGrandCharmRerollCandidate(ctx *context.Status) bool {
	ctx.RefreshGameData()
	items := ctx.Data.Inventory.ByLocation(item.LocationStash, item.LocationSharedStash)
	_, ok := hasItemsForGrandCharmReroll(ctx, items)
	return ok
}

// isGemItem checks if an item is any type of gem (any tier, not just perfect)
func isGemItem(i data.Item) bool {
	itemType := strings.ToLower(i.Type().Code)
	gemTypes := []string{
		strings.ToLower(item.TypeAmethyst),
		strings.ToLower(item.TypeDiamond),
		strings.ToLower(item.TypeEmerald),
		strings.ToLower(item.TypeRuby),
		strings.ToLower(item.TypeSapphire),
		strings.ToLower(item.TypeTopaz),
		strings.ToLower(item.TypeSkull),
	}
	for _, gemType := range gemTypes {
		if itemType == gemType {
			return true
		}
	}
	return false
}

// isEssentialConsumable checks if an item is essential for bot operation (tomes, keys, potions)
func isEssentialConsumable(i data.Item) bool {
	name := string(i.Name)

	// Tomes - bot needs these
	if name == "TomeOfTownPortal" || name == "TomeOfIdentify" {
		return true
	}

	// Regular keys for locked chests (NOT uber keys)
	if name == "Key" {
		return true
	}

	// Potions - bot needs these to survive
	if strings.Contains(name, "Potion") || strings.Contains(name, "potion") {
		return true
	}

	// Scrolls
	if name == "ScrollOfTownPortal" || name == "ScrollOfIdentify" {
		return true
	}

	return false
}
