package webserver

import (
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/bwmarrin/snowflake"
)

func InitiateSnowflakeNode() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatal("Error initializing snowflake %s", err)
	}

	return node
}