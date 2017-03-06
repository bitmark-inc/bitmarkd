package peer

import (
	"github.com/bitmark-inc/bitmarkd/zmqutil"
)

func FetchConnectors() []*zmqutil.Client {
	globalData.RLock()
	defer globalData.RUnlock()
	return globalData.connectorClients
}

func FetchSubscribers() []*zmqutil.Client {
	globalData.RLock()
	defer globalData.RUnlock()
	return globalData.subscriberClients
}
