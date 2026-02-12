package uid

import (
	"github.com/bwmarrin/snowflake"
	"github.com/labstack/gommon/log"
	"sync"
)

var (
	node *snowflake.Node
	once sync.Once
)

func Init(machineID int64) {
	once.Do(func() {
		var err error
		node, err = snowflake.NewNode(machineID)
		if err != nil {
			log.Fatalf("failed to initialize snowflake node: %v", err)
		}
	})
}

func Generate() int64 {
	if node == nil {
		log.Fatalf("uid package not initialized")
	}
	return node.Generate().Int64()
}
