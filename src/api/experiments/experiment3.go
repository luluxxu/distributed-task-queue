package experiments

import (
	"github.com/gin-gonic/gin"
)

func Experiment3(router *gin.Engine) {

	// Reuse experiment1 api endpoints to test experiment 3 logic
	Experiment1(router)
}
