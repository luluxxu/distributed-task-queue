package experiments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	count := 50

	for i := 0; i < count; i++ {
		body, _ := json.Marshal(map[string]string{
			"job_type": "short",
			"payload":  fmt.Sprintf("data-%d", i),
		})

		resp, _ := http.Post("http://localhost:8080/task", "application/json", bytes.NewReader(body))
		fmt.Println("Submitted", i, resp.Status)
	}
}
