package main

import (
	"github.com/gin-gonic/gin"
	experiments "github.com/yourusername/distributed-task-queue/src/api/experiments"
	redis "github.com/yourusername/distributed-task-queue/src/redis"
)

func main() {
	// Initialize Redis
	redis.InitRedis()
	defer redis.CloseRedis()

	// initialize Gin router using Default
	router := gin.Default()

	// Experiment2 includes Experiment1 endpoints + queue status endpoint
	experiments.Experiment2(router)
	// "Run()" attaches router to an http server and start the server
	// router.Run("localhost:8080")
	router.Run(":8080")
}
