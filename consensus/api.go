package consensus

// BlockHeight - return global block height
func BlockHeight() uint64 {
	return globalData.machine.electedHeight
}
