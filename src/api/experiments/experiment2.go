package experiments

import (
	"net/http"

	"github.com/gin-gonic/gin"
	redis "github.com/yourusername/distributed-task-queue/src/redis"
)

func Experiment2(router *gin.Engine) {
	// Reuse experiment1 api endpoints for task submission
	Experiment1(router)
	// Add queue status endpoint for monitoring backlog
	router.GET("/queue/status", getQueueStatus)
}

// getQueueStatus returns the current queue lengths for monitoring backlog
func getQueueStatus(c *gin.Context) {
	fifoLength, err := redis.GetFIFOQueueLength()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get FIFO queue length",
		})
		return
	}

	priorityLength, err := redis.GetPriorityQueueLength()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get priority queue length",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fifo_queue_length":     fifoLength,
		"priority_queue_length": priorityLength,
		"total_backlog":         fifoLength + priorityLength,
	})
}
