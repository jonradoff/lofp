package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/jonradoff/lofp/internal/gameworld"
)

// newTestEngine builds a minimal in-memory GameEngine (no MongoDB) with a single
// mug archetype (1326), matching original/scripts/item1.scr — a LIQCONTAINER
// without OPENABLE (open-top drinking vessel, never lockable/closable).
func newTestEngine() *GameEngine {
	e := &GameEngine{
		items: map[int]*gameworld.ItemDef{
			1326: {
				Number:     1326,
				NameID:     407,
				Type:       "LIQCONTAINER",
				Weight:     1,
				Volume:     2,
				Interior:   10,
				Parameter2: 5,
				Container:  "IN",
				Flags:      []string{"DYEABLE", "CRAFTABLE", "ENCRUSTABLE"},
			},
		},
		nouns:      map[int]string{407: "mug"},
		adjectives: map[int]string{},
		rooms:      map[int]*gameworld.Room{},
	}
	return e
}

func TestMugAlwaysShowsOpen(t *testing.T) {
	e := newTestEngine()
	def := e.items[1326]

	// A freshly-created mug has no State set ("") — it must never display as closed
	// or locked because it has no OPENABLE flag (no lid to close).
	name := e.formatContainerName(def, 0, 0, 0, "")
	if !strings.Contains(name, "(open)") {
		t.Fatalf("expected mug to always show (open), got %q", name)
	}
	if strings.Contains(name, "(closed)") {
		t.Fatalf("mug incorrectly shows (closed): %q", name)
	}
}

func TestSipEmptyMugDoesNotConsumeIt(t *testing.T) {
	e := newTestEngine()
	player := &Player{FirstName: "Tester", Inventory: []InventoryItem{
		{Archetype: 1326}, // empty mug, Val2 == 0
	}}

	result := e.doDrink(context.Background(), player, []string{"mug"})

	if len(player.Inventory) != 1 {
		t.Fatalf("expected mug to remain in inventory, got %d items", len(player.Inventory))
	}
	if len(result.Messages) == 0 || !strings.Contains(strings.ToLower(result.Messages[0]), "empty") {
		t.Fatalf("expected an 'empty' message, got %v", result.Messages)
	}
}

func TestSipFilledMugConsumesLiquidNotContainer(t *testing.T) {
	e := newTestEngine()
	player := &Player{FirstName: "Tester", Inventory: []InventoryItem{
		{Archetype: 1326, Val2: 2, State: "filled"}, // 2 sips of liquid
	}}

	// First sip: liquid decreases, mug stays.
	e.doDrink(context.Background(), player, []string{"mug"})
	if len(player.Inventory) != 1 {
		t.Fatalf("mug was removed from inventory after first sip")
	}
	if player.Inventory[0].Val2 != 1 {
		t.Fatalf("expected 1 sip remaining, got %d", player.Inventory[0].Val2)
	}

	// Second (last) sip: liquid is gone, but the mug itself must remain.
	result := e.doDrink(context.Background(), player, []string{"mug"})
	if len(player.Inventory) != 1 {
		t.Fatalf("mug was removed from inventory after last sip — expected only the liquid to be consumed")
	}
	if player.Inventory[0].Val2 != 0 {
		t.Fatalf("expected 0 sips remaining, got %d", player.Inventory[0].Val2)
	}
	if len(result.Messages) == 0 || !strings.Contains(result.Messages[0], "finish") {
		t.Fatalf("expected a 'finish' message, got %v", result.Messages)
	}
}
