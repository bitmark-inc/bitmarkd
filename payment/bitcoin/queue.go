// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

// An item managed in a priority queue.
type priorityItem struct {
	payId         string // indicates block of transactions
	txId          string // the currency transaction to monitor
	confirmations uint64 // required number of confirmations
	blockNumber   uint64 // the expected block number
	index         int    // index needed for container/heap
}

// A priority queue implements heap.Interface and holds Items.
type priorityQueue []*priorityItem

// number of items in the queue
func (pq priorityQueue) Len() int {
	return len(pq)
}

// to determine the ordering
// need to get smallest block number first
func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].blockNumber < pq[j].blockNumber
}

// for re-ordering the heap
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// add a new item to the heap
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*priorityItem)
	item.index = n
	*pq = append(*pq, item)
}

// take the smallest item from the heap
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// // update modifies the priority and value of an Item in the queue.
// func (pq *priorityQueue) Update(item *Item, value string, priority int) {
// 	item.value = value
// 	item.priority = priority
// 	heap.Fix(pq, item.index)
// }
