package global

import "sync"

var (
	ClientsMap sync.Map //k: remoteaddr v:context
	OnlineClientMap sync.Map //k: uid v:context
)