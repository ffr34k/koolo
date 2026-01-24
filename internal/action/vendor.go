package action

import (
	"fmt"
	"log/slog"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	botCtx "github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/town"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/hectorgimenez/koolo/internal/utils"
	"github.com/lxn/win"
)

// VendorRefillOpts configures vendor refill behavior
type VendorRefillOpts struct {
	ForceRefill    bool     // Force refill even if not needed
	SellJunk       bool     // Sell junk items to vendor
	BuyConsumables bool     // Buy potions, scrolls, keys
	LockConfig     [][]int  // Inventory slots to protect from selling
}

func VendorRefill(opts VendorRefillOpts) (err error) {
	ctx := botCtx.Get()
	ctx.SetLastAction("VendorRefill")

	if !opts.ForceRefill {
		if ctx.Data.PlayerUnit.TotalPlayerGold() <= 100 && ctx.Data.IsLevelingCharacter {
			if lvl, found := ctx.Data.PlayerUnit.FindStat(stat.Level, 0); found && lvl.Value <= 1 {
				return nil
			}
		}
	}

	// Check if we should skip vendor visit
	hasJunkToSell := false
	if opts.SellJunk {
		if len(opts.LockConfig) > 0 {
			hasJunkToSell = len(town.ItemsToBeSold(opts.LockConfig)) > 0
		} else {
			hasJunkToSell = len(town.ItemsToBeSold()) > 0
		}
	}

	// Skip if nothing to do
	if !opts.ForceRefill && !opts.BuyConsumables && !hasJunkToSell {
		return nil
	}
	if !opts.ForceRefill && !hasJunkToSell && !shouldVisitVendor() && len(opts.LockConfig) == 0 {
		return nil
	}

	ctx.Logger.Info("Visiting vendor...", slog.Bool("forceRefill", opts.ForceRefill))

	vendorNPC, err := OpenVendorTradeMenu(opts.BuyConsumables) // only consider key vendor switch if buying
	if err != nil {
		return err
	}

	if opts.SellJunk {
		if len(opts.LockConfig) > 0 {
			town.SellJunk(opts.LockConfig)
		} else {
			town.SellJunk()
		}
	}
	SwitchVendorTab(4)
	ctx.RefreshGameData()

	if opts.BuyConsumables {
		town.BuyConsumables(opts.ForceRefill)
	}

	_ = vendorNPC // used for logging if needed
	return step.CloseAllMenus()
}

// OpenVendorTradeMenu opens the trade menu with the appropriate vendor for the current town.
// If considerKeyVendor is true, it may switch to a different vendor if keys need to be bought.
// Returns the vendor NPC that was used.
func OpenVendorTradeMenu(considerKeyVendor bool) (npc.ID, error) {
	ctx := botCtx.Get()

	currentTown := town.GetTownByArea(ctx.Data.PlayerUnit.Area)
	if currentTown == nil {
		return 0, fmt.Errorf("not in a recognized town")
	}

	vendorNPC := currentTown.RefillNPC()

	// Optionally switch to key-selling vendor if needed
	if considerKeyVendor {
		if vendorNPC == npc.Drognan {
			_, needsBuy := town.ShouldBuyKeys()
			if needsBuy && ctx.Data.PlayerUnit.Class != data.Assassin {
				vendorNPC = npc.Lysander
			}
		}
		if vendorNPC == npc.Ormus {
			_, needsBuy := town.ShouldBuyKeys()
			if needsBuy && ctx.Data.PlayerUnit.Class != data.Assassin {
				if err := FindHratliEverywhere(); err != nil {
					return 0, err
				}
				vendorNPC = npc.Hratli
			}
		}
	}

	if err := InteractNPC(vendorNPC); err != nil {
		return 0, err
	}

	// Jamella (Act 4) trade button is the first one, others need DOWN
	if vendorNPC == npc.Jamella {
		ctx.HID.KeySequence(win.VK_HOME, win.VK_RETURN)
	} else {
		ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_RETURN)
	}

	return vendorNPC, nil
}

func BuyAtVendor(vendor npc.ID, items ...VendorItemRequest) error {
	ctx := botCtx.Get()
	ctx.SetLastAction("BuyAtVendor")

	err := InteractNPC(vendor)
	if err != nil {
		return err
	}

	// Jamella trade button is the first one
	if vendor == npc.Jamella {
		ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_RETURN)
	} else {
		ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_RETURN)
	}

	for _, i := range items {
		SwitchVendorTab(i.Tab)
		itm, found := ctx.Data.Inventory.Find(i.Item, item.LocationVendor)
		if found {
			town.BuyItem(itm, i.Quantity)
		} else {
			ctx.Logger.Warn("Item not found in vendor", slog.String("Item", string(i.Item)))
		}
	}

	return step.CloseAllMenus()
}

type VendorItemRequest struct {
	Item     item.Name
	Quantity int
	Tab      int
}

func shouldVisitVendor() bool {
	ctx := botCtx.Get()
	ctx.SetLastStep("shouldVisitVendor")

	if len(town.ItemsToBeSold()) > 0 {
		return true
	}

	if ctx.Data.PlayerUnit.TotalPlayerGold() < 1000 {
		return false
	}

	if ctx.BeltManager.ShouldBuyPotions() || town.ShouldBuyTPs() || town.ShouldBuyIDs() {
		return true
	}

	return false
}

func SwitchVendorTab(tab int) {
	// Ensure any chat messages that could prevent clicking on the tab are cleared
	ClearMessages()
	utils.Sleep(200)

	ctx := context.Get()
	ctx.SetLastStep("switchVendorTab")

	if ctx.GameReader.LegacyGraphics() {
		x := ui.SwitchVendorTabBtnXClassic
		y := ui.SwitchVendorTabBtnYClassic

		tabSize := ui.SwitchVendorTabBtnTabSizeClassic
		x = x + tabSize*tab - tabSize/2
		if err := ctx.HID.Click(game.LeftButton, x, y); err != nil {
			ctx.Logger.Error("Click failed", "error", err)
		}
		utils.PingSleep(utils.Medium, 500) // Medium operation: Wait for tab switch
	} else {
		x := ui.SwitchVendorTabBtnX
		y := ui.SwitchVendorTabBtnY

		tabSize := ui.SwitchVendorTabBtnTabSize
		x = x + tabSize*tab - tabSize/2
		if err := ctx.HID.Click(game.LeftButton, x, y); err != nil {
			ctx.Logger.Error("Click failed", "error", err)
		}
		utils.PingSleep(utils.Medium, 500) // Medium operation: Wait for tab switch
	}
}
