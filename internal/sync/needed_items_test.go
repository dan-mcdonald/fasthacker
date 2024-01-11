package sync

import "fmt"

func Example_neededItems() {
	// Create a new neededItems
	needed := newNeededItems()

	needed.notifySeen(itemSighting{3, true})
	needed.notifySeen(itemSighting{5, true})
	needed.notifySeen(itemSighting{1, true})
	needed.notifySeen(itemSighting{7, false})

	fmt.Println(needed.size())
	needed.remove(2)
	needed.remove(4)
	needed.remove(6)
	needed.remove(7)
	fmt.Println(needed.size())

	// Output:
	// 4
	// 0
}
