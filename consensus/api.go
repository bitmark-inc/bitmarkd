package consensus

// BlockHeight - return global block height
func BlockHeight() uint64 {
	if nil == globalData.machine {
		return 0
	}
	return globalData.machine.targetHeight
}
