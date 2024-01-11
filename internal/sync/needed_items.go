package sync

import "github.com/dan-mcdonald/fasthacker/internal/model"

type neededItems struct {
	needed         map[model.ItemID]struct{}
	maxKnownItemID model.ItemID
}

func newNeededItems() *neededItems {
	return &neededItems{
		needed:         make(map[model.ItemID]struct{}),
		maxKnownItemID: model.ItemID(0),
	}
}

func (n *neededItems) add(itemID model.ItemID) {
	n.needed[itemID] = struct{}{}
}

func (n *neededItems) remove(itemID model.ItemID) {
	delete(n.needed, itemID)
}

func (n *neededItems) next() model.ItemID {
	for itemID := range n.needed {
		return itemID
	}
	panic("attempted next() when items needed empty")
}

func (n *neededItems) size() int {
	return len(n.needed)
}

func (n *neededItems) empty() bool {
	return n.size() == 0
}

func (n *neededItems) notifySeen(seen itemSighting) {
	if seen.id > n.maxKnownItemID {
		for i := n.maxKnownItemID + 1; i <= seen.id; i++ {
			n.add(i)
		}
		n.maxKnownItemID = seen.id
	}
	if seen.present {
		n.remove(seen.id)
	}
}
