package pkg

import (
	"log"
	"mtgo/data"
	"mtgo/tools"
	"slices"

	"github.com/goccy/go-json"
)

type transfer struct {
	Action string
	Item   string `json:"item"`
	With   string `json:"with"`
	Count  int32  `json:"count"`
}

// QuestAccept updates an existing Accepted quest, or creates and appends new Accepted Quest to cache and Character
func QuestAccept(qid string, sessionID string, event *data.ProfileChangesEvent) {
	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	cachedQuests, err := data.GetQuestCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}
	length := len(cachedQuests.Index)
	time := int(tools.GetCurrentTimeInSeconds())

	query := data.GetQuestFromQueryByID(qid)

	quest, ok := cachedQuests.Index[qid]
	if ok { // if exists, update cache and copy to quest on character
		cachedQuest := character.Quests[quest]

		cachedQuest.Status = "Started"
		cachedQuest.StartTime = time
		cachedQuest.StatusTimers[cachedQuest.Status] = time
	} else {
		quest := &data.CharacterQuest{
			QID:          qid,
			StartTime:    time,
			Status:       "Started",
			StatusTimers: map[string]int{},
		}

		if query.Conditions.AvailableForStart != nil && query.Conditions.AvailableForStart.Quest != nil {
			startCondition := query.Conditions.AvailableForStart.Quest
			for _, questCondition := range startCondition {
				if questCondition.AvailableAfter > 0 {
					quest.StartTime = 0
					quest.Status = "AvailableAfter"
					quest.AvailableAfter = time + questCondition.AvailableAfter
				}
			}
		}

		cachedQuests.Index[qid] = int8(length)
		character.Quests = append(character.Quests, *quest)
	}

	if query.Rewards.Start != nil {
		log.Println("There are rewards heeyrrrr!")
		//log.Println(event.ProfileChanges[character.ID].ID)

		// TODO: Apply then Get Quest rewards and then route messages from there
		// Character.ApplyQuestRewardsToCharacter()  applies the given reward
		// Quests.GetQuestReward() returns the given reward
		// CreateNPCMessageWithReward()
	}

	dialogue, err := data.GetDialogueByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	dialog, message := data.CreateQuestDialogue(character.ID, "QuestStart", query.Trader, query.Dialogue.Description)
	dialog.New++
	dialog.Messages = append(dialog.Messages, *message)

	(*dialogue)[query.Trader] = dialog

	notification := data.CreateNotification(message)

	connection := data.GetConnection(character.ID)
	if connection == nil {
		log.Println("Can't send message to character because connection is nil, storing...")
		storage, err := data.GetStorageByID(character.ID)
		if err != nil {
			log.Println(err)
			return
		}

		storage.Mailbox = append(storage.Mailbox, notification)
		err = storage.SaveStorage(character.ID)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		if err := connection.SendMessage(notification); err != nil {
			return
		}
	}

	//TODO: Get new player quests from data now that we've accepted one
	quests, err := data.GetQuestsAvailableToPlayer(*character)
	if err != nil {
		log.Println(err)
		return
	}

	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		changes.Quests = quests
		event.ProfileChanges.Set(character.ID, changes)
	}
	if err = dialogue.SaveDialogue(character.ID); err != nil {
		return
	}
	if err = character.SaveCharacter(); err != nil {
		return
	}

}

func ApplyQuestRewardsToCharacter(rewards *data.QuestRewards) {
	log.Println()
}

type examine struct {
	Action    string     `json:"Action"`
	Item      string     `json:"item"`
	FromOwner *fromOwner `json:"fromOwner,omitempty"`
}
type fromOwner struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func ExamineItem(action map[string]any, sessionID string) {
	examine := new(examine)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &examine); err != nil {
		log.Println(err)
		return
	}
	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	var item *data.DatabaseItem
	if examine.FromOwner == nil {
		log.Println("Examining Item from Player Inventory")
		cache, err := data.GetInventoryCacheByID(character.ID)
		if err != nil {
			log.Println(err)
			return
		}

		if index := cache.GetIndexOfItemByID(examine.Item); index != nil {
			itemInInventory := character.Inventory.Items[*index]
			item, err = data.GetItemByID(itemInInventory.TPL)
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			log.Println("[EXAMINE] Examining Item", examine.Item, " from Player Inventory failed, does not exist!")
			return
		}
	} else {
		switch examine.FromOwner.Type {
		case "Trader":
			trader, err := data.GetTraderByUID(examine.FromOwner.ID)
			if err != nil {
				log.Println(err)
				return
			}

			assortItem := trader.GetAssortItemByID(examine.Item)
			item, err = data.GetItemByID(assortItem[0].Tpl)
			if err != nil {
				log.Println(err)
				return
			}

		case "HideoutUpgrade":
		case "HideoutProduction":
		case "ScavCase":
			item, err = data.GetItemByID(examine.Item)
			if err != nil {
				log.Println(err)
				return
			}

		case "RagFair":
		default:
			log.Println("[EXAMINE] FromOwner.Type: ", examine.FromOwner.Type, "is not supported, returning...")
			return
		}
	}

	if item == nil {
		log.Println("[EXAMINE] Examining Item", examine.Item, "failed, does not exist in Item Database")
		return
	}

	character.Encyclopedia[item.ID] = true
	log.Println("[EXAMINE] Encyclopedia entry added for", item.ID)

	//add experience
	experience, ok := item.Props["ExamineExperience"].(float64)
	if !ok {
		log.Println("[EXAMINE] Item", examine.Item, "does not have ExamineExperience property, returning...")
		return
	}

	character.Info.Experience += int32(experience)
}

type move struct {
	Action string
	Item   string `json:"item"`
	To     moveTo `json:"to"`
}

type moveTo struct {
	ID        string          `json:"id"`
	Container string          `json:"container"`
	Location  *moveToLocation `json:"location,omitempty"`
}

type moveToLocation struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	R          string  `json:"r"`
	IsSearched bool    `json:"isSearched"`
}

func MoveItemInStash(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	move := new(move)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &move); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	cache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	index := cache.GetIndexOfItemByID(move.Item)
	itemInInventory := &character.Inventory.Items[*index]

	if move.To.Location == nil {
		if move.To.Container == "cartridges" {
			//TODO: fix this, this is terrible!
			counter := 0
			for _, item := range character.Inventory.Items {
				if item.ParentID != move.To.ID {
					continue
				}
				counter++
			}

			itemInInventory.Location = counter
		} else {
			itemInInventory.Location = move.To.Location //nil
		}
	} else {
		moveToLocation := move.To.Location

		var rotation float64 = 0
		if moveToLocation.R == "Vertical" || moveToLocation.R == "1" {
			rotation++
		}

		itemInInventory.Location = data.InventoryItemLocation{
			IsSearched: moveToLocation.IsSearched,
			R:          rotation,
			X:          moveToLocation.X,
			Y:          moveToLocation.Y,
		}
	}

	itemInInventory.ParentID = move.To.ID
	itemInInventory.SlotID = move.To.Container

	_, ok := cache.Stash.Container.FlatMap[move.Item]
	if ok || itemInInventory.SlotID != "hideout" {
		cache.ClearItemFromContainerMap(move.Item)
	}

	if itemInInventory.Location != nil {
		if _, ok := itemInInventory.Location.(int); !ok {
			cache.SetNewItemFlatMap([]data.InventoryItem{*itemInInventory})
			cache.AddItemToContainerMap(move.Item)
		}
	}

	//cache.SetInventoryIndex(&character.Inventory)
	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		changes.Production = nil
		event.ProfileChanges.Set(character.ID, changes)
	}
}

type swap struct {
	Action string
	Item   string `json:"item"`
	To     moveTo `json:"to"`
	Item2  string `json:"item2"`
	To2    moveTo `json:"to2"`
}

func SwapItemInStash(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	swap := new(swap)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &swap); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	cache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	index := *cache.GetIndexOfItemByID(swap.Item)
	itemInInventory := &character.Inventory.Items[index]

	if swap.To.Location == nil {
		if swap.To.Container == "cartridges" {
			//TODO: fix this, this is terrible!
			counter := 0
			for _, item := range character.Inventory.Items {
				if item.ParentID != swap.To.ID {
					continue
				}
				counter++
			}

			itemInInventory.Location = counter
		} else {
			itemInInventory.Location = nil
		}
	} else {
		moveToLocation := swap.To.Location

		var rotation float64 = 0
		if moveToLocation.R == "Vertical" || moveToLocation.R == "1" {
			rotation++
		}

		itemInInventory.Location = data.InventoryItemLocation{
			IsSearched: moveToLocation.IsSearched,
			R:          rotation,
			X:          moveToLocation.X,
			Y:          moveToLocation.Y,
		}
	}

	itemInInventory.ParentID = swap.To.ID
	itemInInventory.SlotID = swap.To.Container

	index = cache.Lookup.Forward[swap.Item2]
	itemInInventory = &character.Inventory.Items[index]

	if swap.To2.Location != nil {
		moveToLocation := swap.To2.Location
		var rotation float64 = 0
		if moveToLocation.R == "Vertical" || moveToLocation.R == "1" {
			rotation++
		}

		itemInInventory.Location = data.InventoryItemLocation{
			IsSearched: moveToLocation.IsSearched,
			R:          rotation,
			X:          moveToLocation.X,
			Y:          moveToLocation.Y,
		}
	} else {
		itemInInventory.Location = nil
	}

	itemInInventory.ParentID = swap.To2.ID
	itemInInventory.SlotID = swap.To2.Container

	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		changes.Production = nil
		event.ProfileChanges.Set(character.ID, changes)
	}
}

type foldItem struct {
	Action string
	Item   string `json:"item"`
	Value  bool   `json:"value"`
}

func FoldItem(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	fold := new(foldItem)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &fold); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	inventoryCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	index := inventoryCache.GetIndexOfItemByID(fold.Item)
	if index == nil {
		log.Println("Item", fold.Item, "does not exist in cache!")
		return
	}
	itemInInventory := &character.Inventory.Items[*index]
	if itemInInventory.UPD == nil || itemInInventory.UPD.Foldable == nil {
		log.Println(itemInInventory.ID, "cannot be folded!")
		return
	}

	itemInInventory.UPD.Foldable.Folded = fold.Value

	inventoryCache.ResetItemSizeInContainer(itemInInventory, &character.Inventory)
	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		changes.Production = nil
		event.ProfileChanges.Set(character.ID, changes)
	}
}

type readEncyclopedia struct {
	Action string   `json:"Action"`
	IDs    []string `json:"ids"`
}

func ReadEncyclopedia(action map[string]any, sessionID string) {
	readEncyclopedia := new(readEncyclopedia)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &readEncyclopedia); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	for _, id := range readEncyclopedia.IDs {
		character.Encyclopedia[id] = true
	}
}

type merge struct {
	Action string
	Item   string `json:"item"`
	With   string `json:"with"`
}

func MergeItem(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	merge := new(merge)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &merge); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	inventoryCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	toMergeIndex := *inventoryCache.GetIndexOfItemByID(merge.Item)
	toMerge := &character.Inventory.Items[toMergeIndex]

	mergeWithIndex := *inventoryCache.GetIndexOfItemByID(merge.With)
	mergeWith := character.Inventory.Items[mergeWithIndex]

	mergeWith.UPD.StackObjectsCount += toMerge.UPD.StackObjectsCount

	inventoryCache.ClearItemFromContainer(toMerge.ID)
	character.Inventory.RemoveSingleItemFromInventoryByIndex(toMergeIndex)
	inventoryCache.SetInventoryIndex(&character.Inventory)

	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		changes.Items.Change = append(changes.Items.Change, mergeWith)
		changes.Items.Del = append(changes.Items.Del, data.InventoryItem{ID: toMerge.ID})
		event.ProfileChanges.Set(character.ID, changes)
	}
}

func TransferItem(action map[string]any, sessionID string) {
	transfer := new(transfer)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &transfer); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	inventoryCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	toMergeIndex := *inventoryCache.GetIndexOfItemByID(transfer.Item)
	toMerge := &character.Inventory.Items[toMergeIndex]

	mergeWithIndex := *inventoryCache.GetIndexOfItemByID(transfer.With)
	mergeWith := &character.Inventory.Items[mergeWithIndex]

	toMerge.UPD.StackObjectsCount -= transfer.Count
	mergeWith.UPD.StackObjectsCount += transfer.Count
}

type split struct {
	Action    string `json:"Action"`
	SplitItem string `json:"splitItem"`
	NewItem   string `json:"newItem"`
	Container moveTo `json:"container"`
	Count     int32  `json:"count"`
}

func SplitItem(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	split := new(split)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &split); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	invCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	originalItem := &character.Inventory.Items[*invCache.GetIndexOfItemByID(split.SplitItem)]
	originalItem.UPD.StackObjectsCount -= split.Count

	newItem := originalItem.Clone()
	newItem.ID = split.NewItem
	newItem.ParentID = split.Container.ID
	newItem.SlotID = split.Container.Container

	newItem.UPD.StackObjectsCount = split.Count

	if split.Container.Location == nil {
		if split.Container.Container == "cartridges" {
			//TODO: fix this, this is terrible!
			counter := 0
			for _, item := range character.Inventory.Items {
				if item.ParentID != split.Container.ID {
					continue
				}
				counter++
			}

			newItem.Location = counter
		} else {
			newItem.Location = nil
		}
	} else {
		location := data.InventoryItemLocation{
			IsSearched: split.Container.Location.IsSearched,
			X:          split.Container.Location.X,
			Y:          split.Container.Location.Y,
			R:          1,
		}

		if split.Container.Location.R == "Vertical" {
			location.R = float64(1)
		} else {
			location.R = float64(0)
		}

		newItem.Location = location

		height, width := data.MeasurePurchaseForInventoryMapping([]data.InventoryItem{*newItem})
		itemFlatMap := invCache.CreateFlatMapLookup(height, width, newItem)
		itemFlatMap.Coordinates = invCache.GenerateCoordinatesFromLocation(*itemFlatMap)
		invCache.AddItemToContainer(split.NewItem, itemFlatMap)
	}

	character.Inventory.Items = append(character.Inventory.Items, *newItem)
	invCache.SetSingleInventoryIndex(newItem.ID, int16(len(character.Inventory.Items)-1))

	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		//changes.Items.Change = append(changes.Items.Change, *originalItem)
		changes.Items.New = append(changes.Items.New, data.InventoryItem{ID: newItem.ID, TPL: newItem.TPL, UPD: newItem.UPD})
		event.ProfileChanges.Set(character.ID, changes)
	}
}

type remove struct {
	Action string `json:"Action"`
	ItemID string `json:"item"`
}

func RemoveItem(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	remove := new(remove)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &remove); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	inventoryCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}
	itemChildren := data.GetInventoryItemFamilyTreeIDs(character.Inventory.Items, remove.ItemID)

	var itemIndex int16
	toDelete := make([]int16, 0, len(itemChildren))
	for _, itemID := range itemChildren {
		itemIndex = *inventoryCache.GetIndexOfItemByID(itemID)
		toDelete = append(toDelete, itemIndex)
	}

	inventoryCache.ClearItemFromContainer(remove.ItemID)
	character.Inventory.RemoveItemsFromInventoryByIndices(toDelete)
	inventoryCache.SetInventoryIndex(&character.Inventory)
	if changes, ok := event.ProfileChanges.Get(character.ID); ok {
		changes.Items.Del = append(changes.Items.Del, data.InventoryItem{ID: remove.ItemID})
		event.ProfileChanges.Set(character.ID, changes)
	}
}

type applyInventoryChanges struct {
	Action       string
	ChangedItems []any `json:"changedItems"`
}

//TODO: Make ApplyInventoryChanges not look like shit

func ApplyInventoryChanges(action map[string]any, sessionID string) {
	applyInventoryChanges := new(applyInventoryChanges)
	input, _ := json.MarshalNoEscape(action)
	if err := json.UnmarshalNoEscape(input, &applyInventoryChanges); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	cache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	for _, item := range applyInventoryChanges.ChangedItems {
		properties, ok := item.(map[string]any)
		if !ok {
			log.Println("Cannot type assert item from Auto-Sort items slice")
			return
		}

		UID, ok := properties["_id"].(string)
		if !ok {
			log.Println("Cannot type assert item `_id` property from Auto-Sort items slice")
			return
		}
		itemInInventory := &character.Inventory.Items[*cache.GetIndexOfItemByID(UID)]

		parent, ok := properties["parentId"].(string)
		if !ok {
			log.Println("Cannot type assert item `parentId` property from Auto-Sort items slice")
			return
		}
		itemInInventory.ParentID = parent

		slotID, ok := properties["slotId"].(string)
		if !ok {
			log.Println("Cannot type assert item `slotId` property from Auto-Sort items slice")
			return
		}
		itemInInventory.SlotID = slotID

		if location, ok := properties["location"].(float64); ok {
			itemInInventory.Location = location
			continue
		}

		location, ok := properties["location"].(map[string]any)
		if !ok {
			itemInInventory.Location = nil
			continue
		}

		r, ok := location["r"].(string)
		if !ok {
			log.Println("Cannot type assert item.Location `r` property from Auto-Sort items slice")
			return
		}

		itemLocation := data.InventoryItemLocation{
			IsSearched: false,
			R:          nil,
			X:          nil,
			Y:          nil,
		}

		if r == "Horizontal" || r == "1" {
			itemLocation.R = float64(0)
		} else {
			itemLocation.R = float64(1)
		}

		if x, ok := location["r"].(float64); ok {
			itemLocation.X = x
		}

		if isSearched, ok := location["isSearched"].(bool); ok {
			itemLocation.IsSearched = isSearched
		}

		if y, ok := location["r"].(float64); ok {
			itemLocation.Y = y
		}

		itemInInventory.Location = itemLocation
	}
	cache.SetInventoryIndex(&character.Inventory)
	cache.SetInventoryStash(&character.Inventory)
}

type buyFrom struct {
	Action      string
	Type        string          `json:"type"`
	TID         string          `json:"tid"`
	ItemID      string          `json:"item_id"`
	Count       int32           `json:"count"`
	SchemeID    int8            `json:"scheme_id"`
	SchemeItems []tradingScheme `json:"scheme_items"`
}

type tradingScheme struct {
	ID    string
	Count int32
}

type sellTo struct {
	Action string
	Type   string      `json:"type"`
	TID    string      `json:"tid"`
	Items  []soldItems `json:"items"`
	Price  int32       `json:"price"`
}

type soldItems struct {
	ID       string `json:"id"`
	Count    int32  `json:"count"`
	SchemeID int8   `json:"scheme_id"`
}

func TradingConfirm(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Println(err)
		return
	}

	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}

	switch action["type"] {
	case "buy_from_trader":
		buy := new(buyFrom)
		if err := json.UnmarshalNoEscape(input, &buy); err != nil {
			log.Println(err)
			return
		}
		buyFromTrader(buy, character, event)
	case "sell_to_trader":
		sell := new(sellTo)
		if err := json.UnmarshalNoEscape(input, &sell); err != nil {
			log.Println(err)
			return
		}
		sellToTrader(sell, character, event)
	default:
		log.Println("YO! TRADINGCONFIRM.", action["type"], "ISNT SUPPORTED YET HAHAHHAHAHAHAHAHHAHAHHHHHHHHHHHHHAHAHAHAHAHHAHA")
		return
	}
}

func buyFromTrader(tradeConfirm *buyFrom, character *data.Character[map[string]data.PlayerTradersInfo], event *data.ProfileChangesEvent) {
	invCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	trader, err := data.GetTraderByUID(tradeConfirm.TID)
	if err != nil {
		log.Println(err)
		return
	}

	assortItem := trader.GetAssortItemByID(tradeConfirm.ItemID)
	if assortItem == nil {
		log.Println("Item of", tradeConfirm.ItemID, "does not exist in trader assort, killing!")
		return
	}

	inventoryItems := data.ConvertAssortItemsToInventoryItem(assortItem, &character.Inventory.Stash)
	if len(inventoryItems) == 0 {
		log.Println("Converting Assort Item to Inventory Item failed, killing")
		return
	}

	item, err := data.GetItemByID(inventoryItems[len(inventoryItems)-1].TPL)
	if err != nil {
		log.Println(err)
		return
	}

	stackMaxSize := item.GetStackMaxSize()
	stackSlice := GetCorrectAmountOfItemsPurchased(tradeConfirm.Count, stackMaxSize)
	// Basically gets the correct amount of items to be created, based on StackSize

	// Create copy-of Character.Inventory.Items for modification in the case of any failures to assign later
	copyOfItems := make([]data.InventoryItem, 0, len(character.Inventory.Items)+(len(inventoryItems)*len(stackSlice)))
	copyOfItems = append(copyOfItems, character.Inventory.Items...)
	// Create copy-of invCache.Stash.Container for modification in the case of any failures to assign later
	copyOfMap := invCache.Stash.Container

	toAdd := make([]data.InventoryItem, 0, len(stackSlice))

	height, width := data.MeasurePurchaseForInventoryMapping(inventoryItems)
	var copyOfInventoryItems []data.InventoryItem

	for _, stack := range stackSlice {
		copyOfInventoryItems = data.AssignNewIDs(inventoryItems)

		index := len(copyOfInventoryItems) - 1
		mainItem := copyOfInventoryItems[index]

		validLocation := invCache.GetValidLocationForItem(height, width)
		if validLocation == nil {
			log.Println("Item", tradeConfirm.ItemID, "was not purchased because we could not find a position in your inventory!!")
			invCache.Stash.Container = copyOfMap //if failure, assign old map
			return
		}

		if stackMaxSize > 1 {
			mainItem.UPD.StackObjectsCount = stack
		}

		mainItem.Location = &data.InventoryItemLocation{
			IsSearched: true,
			R:          float64(0),
			X:          float64(validLocation.X),
			Y:          float64(validLocation.Y),
		}

		itemFlatMap := invCache.CreateFlatMapLookup(height, width, &mainItem)
		itemFlatMap.Coordinates = validLocation.MapInfo
		invCache.AddItemToContainer(mainItem.ID, itemFlatMap)

		copyOfInventoryItems[index] = mainItem
		toAdd = append(toAdd, copyOfInventoryItems...)
	}
	copyOfInventoryItems = nil
	changes, ok := event.ProfileChanges.Get(character.ID)
	if !ok {
		log.Fatal("profile changes event does not exist")
	}

	toDelete := make(map[string]int16)
	traderRelations := character.TradersInfo[tradeConfirm.TID]
	for _, scheme := range tradeConfirm.SchemeItems {
		index := invCache.GetIndexOfItemByID(scheme.ID)
		if index == nil {
			log.Println("Index of", scheme.ID, "does not exist in cache, killing!")
			return
		}

		itemInInventory := copyOfItems[*index]

		currency := *data.GetCurrencyByName(trader.Base.Currency)
		if data.IsCurrencyByID(itemInInventory.TPL) {
			traderRelations.SalesSum += float32(scheme.Count)
		} else {
			priceOfItem, err := data.GetPriceByID(itemInInventory.TPL)
			if err != nil {
				log.Println(err)
				return
			}

			if trader.Base.Currency != "RUB" {
				if conversion, err := data.ConvertFromRouble(priceOfItem, currency); err == nil {
					traderRelations.SalesSum += float32(conversion)
				} else {
					log.Println(err)
					return
				}
			} else {
				traderRelations.SalesSum += float32(priceOfItem)
			}
		}

		if itemInInventory.UPD != nil && itemInInventory.UPD.StackObjectsCount != 0 {
			var remainingBalance = scheme.Count

			if itemInInventory.UPD.StackObjectsCount > remainingBalance {
				itemInInventory.UPD.StackObjectsCount -= remainingBalance

				changes.Items.Change = append(changes.Items.Change, itemInInventory)
			} else if itemInInventory.UPD.StackObjectsCount == remainingBalance {
				toDelete[itemInInventory.ID] = *index
			} else {
				remainingBalance -= itemInInventory.UPD.StackObjectsCount

				toDelete[itemInInventory.ID] = *index

				//TODO: Consider creating a look-up cache for mergable Inventory.Items

				var toChange []data.InventoryItem
				for idx, item := range copyOfItems {
					if _, ok := toDelete[item.ID]; ok || item.TPL != currency {
						continue
					}

					change := item.UPD.StackObjectsCount - remainingBalance
					if change > 0 {
						remainingBalance -= item.UPD.StackObjectsCount
						toDelete[item.ID] = int16(idx)
						continue
					} else if change == 0 {
						remainingBalance -= item.UPD.StackObjectsCount
						toDelete[item.ID] = int16(idx)
						break
					}

					item.UPD.StackObjectsCount = change
					toChange = append(toChange, item)
					break
				}
				if remainingBalance > 0 {
					log.Println("Insufficient funds to purchase item, returning")
					invCache.Stash.Container = copyOfMap
					return
				}

				changes.Items.Change = append(changes.Items.Change, toChange...)
			}
		} else {
			toDelete[itemInInventory.ID] = *index
		}
	}

	// Add all items from toAdd to Copy of Inventory.Items
	if len(toAdd) == 0 {
		log.Println("balls")
		return
	}

	copyOfItems = append(copyOfItems, toAdd...)
	changes.Items.New = append(changes.Items.New, toAdd...)
	character.Inventory.Items = copyOfItems

	if len(toDelete) != 0 {
		indices := make([]int16, 0, len(toDelete))
		for id, idx := range toDelete {
			invCache.ClearItemFromContainer(id)
			indices = append(indices, idx)

			changes.Items.Del = append(changes.Items.Del, data.InventoryItem{ID: id})
		}
		character.Inventory.RemoveItemsFromInventoryByIndices(indices)
	}
	invCache.SetInventoryIndex(&character.Inventory)

	changes.TraderRelations[tradeConfirm.TID] = traderRelations
	character.TradersInfo[tradeConfirm.TID] = traderRelations

	event.ProfileChanges.Set(character.ID, changes)
	log.Println(len(stackSlice), "of Item", tradeConfirm.ItemID, "purchased!")
}

func sellToTrader(tradeConfirm *sellTo, character *data.Character[map[string]data.PlayerTradersInfo], event *data.ProfileChangesEvent) {
	invCache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	var saleCurrency string
	if trader, err := data.GetTraderByUID(tradeConfirm.TID); err != nil {
		log.Println(err)
		return
	} else {
		saleCurrency = *data.GetCurrencyByName(trader.Base.Currency)
	}

	var stackMaxSize int32
	if item, err := data.GetItemByID(saleCurrency); err != nil {
		log.Println(err)
		return
	} else {
		stackMaxSize = item.GetStackMaxSize()
	}

	cache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	remainingBalance := tradeConfirm.Price

	toChange := make([]data.InventoryItem, 0)
	copyOfItems := make([]data.InventoryItem, 0, len(character.Inventory.Items))
	copyOfItems = append(copyOfItems, character.Inventory.Items...)
	for _, item := range copyOfItems {
		if remainingBalance == 0 {
			break
		}

		if item.TPL != saleCurrency || item.UPD.StackObjectsCount == stackMaxSize {
			continue
		}

		if item.UPD.StackObjectsCount+remainingBalance > stackMaxSize {
			remainingBalance -= stackMaxSize - item.UPD.StackObjectsCount
			item.UPD.StackObjectsCount = stackMaxSize

			toChange = append(toChange, item)
			continue
		} else {
			item.UPD.StackObjectsCount += remainingBalance
			remainingBalance = 0
			toChange = append(toChange, item)
			break
		}
	}

	changes, ok := event.ProfileChanges.Get(character.ID)
	if !ok {
		log.Fatal("profile changes event does not exist")
	}

	if remainingBalance != 0 {
		var toAdd []data.InventoryItem
		copyOfMap := invCache.Stash.Container

		//log.Println("If a new stack isn't made, we cry")

		stackSlice := GetCorrectAmountOfItemsPurchased(remainingBalance, stackMaxSize)
		item := []data.InventoryItem{*data.CreateNewItem(saleCurrency, character.Inventory.Stash)}
		// since it's one item, just get the height and width once
		height, width := data.MeasurePurchaseForInventoryMapping(item)

		for _, stack := range stackSlice {
			mainItem := data.AssignNewIDs(item)[0]

			validLocation := invCache.GetValidLocationForItem(height, width)
			if validLocation == nil {
				log.Println("Item", mainItem.ID, "was not created because we could not find a position in your inventory!")
				invCache.Stash.Container = copyOfMap //if failure, assign old map
				return
			}

			mainItem.UPD.StackObjectsCount = stack
			mainItem.Location = &data.InventoryItemLocation{
				IsSearched: true,
				R:          float64(0),
				X:          float64(validLocation.X),
				Y:          float64(validLocation.Y),
			}

			itemFlatMap := invCache.CreateFlatMapLookup(height, width, &mainItem)
			itemFlatMap.Coordinates = validLocation.MapInfo
			invCache.AddItemToContainer(mainItem.ID, itemFlatMap)

			toAdd = append(toAdd, mainItem)
		}

		copyOfItems = append(copyOfItems, toAdd...)
		changes.Items.New = append(changes.Items.New, toAdd...)
	}

	changes.Items.Change = append(changes.Items.Change, toChange...)
	character.Inventory.Items = copyOfItems

	toDelete := make(map[string]int16)
	for _, item := range tradeConfirm.Items {
		index := *cache.GetIndexOfItemByID(item.ID)
		toDelete[item.ID] = index
	}

	if len(toDelete) != 0 {
		indices := make([]int16, 0, len(toDelete))
		for id, idx := range toDelete {
			invCache.ClearItemFromContainer(id)
			indices = append(indices, idx)
			if _, ok := toDelete[character.Inventory.Items[idx].ParentID]; ok {
				continue
			}
			changes.Items.Del = append(changes.Items.Del, data.InventoryItem{ID: id})
		}
		character.Inventory.RemoveItemsFromInventoryByIndices(indices)
	}
	invCache.SetInventoryIndex(&character.Inventory)

	traderRelations := character.TradersInfo[tradeConfirm.TID]
	traderRelations.SalesSum += float32(tradeConfirm.Price)

	changes.TraderRelations[tradeConfirm.TID] = traderRelations
	character.TradersInfo[tradeConfirm.TID] = traderRelations

	event.ProfileChanges.Set(character.ID, changes)
}

type buyCustomization struct {
	Action string           `json:"Action"`
	Offer  string           `json:"offer"`
	Items  []map[string]any `json:"items"`
}

func CustomizationBuy(action map[string]any, sessionID string) {
	customizationBuy := new(buyCustomization)
	input, _ := json.MarshalNoEscape(action)
	err := json.UnmarshalNoEscape(input, &customizationBuy)
	if err != nil {
		log.Println(err)
		return
	}

	trader, err := data.GetTraderByName("Ragman")
	if err != nil {
		log.Println(err)
		return
	}
	suitsIndex, ok := trader.Index.Suits[customizationBuy.Offer]
	if !ok {
		log.Println("Suit doesn't exist")
		return
	}
	suitID := trader.Suits[suitsIndex].SuiteID

	storage, err := data.GetStorageByID(sessionID)
	if err != nil {
		log.Println(err)
		return
	}

	if !slices.Contains(storage.Suites, suitID) {
		//TODO: Pay for suite before appending to profile
		if len(customizationBuy.Items) == 0 {
			storage.Suites = append(storage.Suites, suitID)
			err = storage.SaveStorage(sessionID)
			if err != nil {
				return
			}
			return
		}
		log.Println("Cannot purchase clothing because I haven't implemented it yet lol")
		return
	}
	log.Println("Clothing was already purchased")
}

type wearCustomization struct {
	Action string   `json:"Action"`
	Suites []string `json:"suites"`
}

const (
	lowerParentID = "5cd944d01388ce000a659df9"
	upperParentID = "5cd944ca1388ce03a44dc2a4"
)

func CustomizationWear(action map[string]any, sessionID string) {
	customizationWear := new(wearCustomization)
	input, _ := json.MarshalNoEscape(action)
	err := json.UnmarshalNoEscape(input, &customizationWear)
	if err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	for _, SID := range customizationWear.Suites {
		customization, err := data.GetCustomizationByID(SID)
		if err != nil {
			log.Println(err)
			return
		}

		parentID := customization.Parent

		if parentID == lowerParentID {
			character.Customization.Feet = customization.Props.Feet
			continue
		}

		if parentID == upperParentID {
			character.Customization.Body = customization.Props.Body
			character.Customization.Hands = customization.Props.Hands
			continue
		}
	}
}

type hideoutUpgrade struct {
	Action    string
	AreaType  int8            `json:"areaType"`
	Items     []tradingScheme `json:"items"`
	TimeStamp float64         `json:"timeStamp"`
}

func HideoutUpgrade(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {
	log.Println("HideoutUpgrade")
	upgrade := new(hideoutUpgrade)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}

	// character:= data.GetCharacterByID(sessionID)
	if err := json.UnmarshalNoEscape(input, &upgrade); err != nil {
		log.Println(err)
		return
	}

	hideoutArea := data.GetHideoutAreaByAreaType(upgrade.AreaType)

	log.Println(hideoutArea)
}

type bindItem struct {
	Action string
	Item   string `json:"item"`
	Index  string `json:"index"`
}

func BindItem(action map[string]any, sessionID string) {
	bind := new(bindItem)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &bind); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	if _, ok := character.Inventory.FastPanel[bind.Index]; !ok {
		character.Inventory.FastPanel[bind.Index] = bind.Item
		return
	}

	if character.Inventory.FastPanel[bind.Index] == bind.Item {
		character.Inventory.FastPanel[bind.Index] = ""
	} else {
		character.Inventory.FastPanel[bind.Index] = bind.Item
	}

}

type tagItem struct {
	Action   string
	Item     string `json:"item"`
	TagName  string `json:"TagName"`
	TagColor string `json:"TagColor"`
}

func TagItem(action map[string]any, sessionID string) {
	tag := new(tagItem)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &tag); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	cache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	index := *cache.GetIndexOfItemByID(tag.Item)
	if character.Inventory.Items[index].UPD != nil {
		if character.Inventory.Items[index].UPD.Tag != nil {
			character.Inventory.Items[index].UPD.Tag.Color = tag.TagColor
			character.Inventory.Items[index].UPD.Tag.Name = tag.TagName
			return
		}
		character.Inventory.Items[index].UPD.Tag = new(data.Tag)
		character.Inventory.Items[index].UPD.Tag.Color = tag.TagColor
		character.Inventory.Items[index].UPD.Tag.Name = tag.TagName
	} else {
		character.Inventory.Items[index].UPD = new(data.ItemUpdate)
		character.Inventory.Items[index].UPD.Tag = new(data.Tag)
		character.Inventory.Items[index].UPD.Tag.Color = tag.TagColor
		character.Inventory.Items[index].UPD.Tag.Name = tag.TagName
	}
}

type toggleItem struct {
	Action string
	Item   string `json:"item"`
	Value  bool   `json:"value"`
}

func ToggleItem(action map[string]any, sessionID string) {
	toggle := new(toggleItem)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &toggle); err != nil {
		log.Println(err)
		return
	}

	character, err := data.GetCharacterByID(sessionID)
	if err != nil {
		log.Fatalln(err)
	}
	cache, err := data.GetInventoryCacheByID(character.ID)
	if err != nil {
		log.Println(err)
		return
	}

	index := *cache.GetIndexOfItemByID(toggle.Item)
	if character.Inventory.Items[index].UPD == nil {
		if character.Inventory.Items[index].UPD.Togglable != nil {
			character.Inventory.Items[index].UPD.Togglable.On = toggle.Value
			return
		}

		character.Inventory.Items[index].UPD.Togglable = new(data.Toggle)
		character.Inventory.Items[index].UPD.Togglable.On = toggle.Value
	} else {
		character.Inventory.Items[index].UPD = new(data.ItemUpdate)
		character.Inventory.Items[index].UPD.Togglable = new(data.Toggle)
		character.Inventory.Items[index].UPD.Togglable.On = toggle.Value
	}
}

type hideoutUpgradeComplete struct {
	Action    string
	AreaType  uint8   `json:"areaType"`
	TimeStamp float64 `json:"timeStamp"`
}

func HideoutUpgradeComplete(action map[string]any, _ string, _ *data.ProfileChangesEvent) {
	log.Println("HideoutUpgradeComplete")
	upgradeComplete := new(hideoutUpgradeComplete)
	input, err := json.MarshalNoEscape(action)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.UnmarshalNoEscape(input, &upgradeComplete); err != nil {
		log.Println(err)
		return
	}

	// character := data.GetCharacterByID(sessionID)
	log.Println(upgradeComplete)
}

func Insure(action map[string]any, sessionID string, event *data.ProfileChangesEvent) {

}
