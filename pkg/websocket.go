package pkg

import (
	"fmt"
	"mtgo/data"
)

const getWebsocket string = "%s/getwebsocket/%s"

func GetWebSocket(sessionID string) string {
	return fmt.Sprintf(getWebsocket, data.GetWebSocketAddress(), sessionID)
}
