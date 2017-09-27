package peer

import (
	"github.com/bitmark-inc/bitmarkd/zmqutil"
)

func FetchConnectors() []*zmqutil.Connected {

	globalData.RLock()

	result := make([]*zmqutil.Connected, 0, len(globalData.connectorClients))

	for _, c := range globalData.connectorClients {
		if nil != c {
			connect := c.ConnectedTo()
			if nil != connect {
				result = append(result, connect)
			}
		}
	}

	globalData.RUnlock()

	return result
}

func FetchSubscribers() []*zmqutil.Connected {

	globalData.RLock()

	result := make([]*zmqutil.Connected, 0, len(globalData.subscriberClients))

	for _, c := range globalData.subscriberClients {
		if nil != c {
			connect := c.ConnectedTo()
			if nil != connect {
				result = append(result, connect)
			}
		}
	}

	globalData.RUnlock()

	return result
}
