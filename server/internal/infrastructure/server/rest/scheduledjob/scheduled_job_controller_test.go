package scheduledjob

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestResolveQueueCode(t *testing.T) {
	// 1. Both provided: prefers queueCode
	assert.Equal(t, "primary-queue", resolveQueueCode("primary-queue", "fallback-queue"))

	// 2. Only queueCode provided
	assert.Equal(t, "primary-queue", resolveQueueCode("primary-queue", ""))

	// 3. Only targetCode provided (fallback)
	assert.Equal(t, "fallback-queue", resolveQueueCode("", "fallback-queue"))

	// 4. Neither provided
	assert.Equal(t, "", resolveQueueCode("", ""))
}

func TestScheduledJobRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectStatus   int
		expectQueueVal string
	}{
		{
			name: "valid with queueCode",
			payload: map[string]interface{}{
				"queueCode":  "orders-queue",
				"vnamespace": "default",
			},
			expectStatus:   http.StatusOK,
			expectQueueVal: "orders-queue",
		},
		{
			name: "valid with targetCode fallback",
			payload: map[string]interface{}{
				"targetCode": "emails-queue",
				"vnamespace": "default",
			},
			expectStatus:   http.StatusOK,
			expectQueueVal: "emails-queue",
		},
		{
			name: "missing both queueCode and targetCode returns 400",
			payload: map[string]interface{}{
				"vnamespace": "default",
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "missing vnamespace returns 400",
			payload: map[string]interface{}{
				"queueCode": "orders-queue",
			},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/test-one-off", func(c *gin.Context) {
				var req createOneOffScheduledJobRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
					return
				}

				queueCode := resolveQueueCode(req.QueueCode, req.TargetCode)
				if queueCode == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "queueCode is required"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"queueCode": queueCode})
			})

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/test-one-off", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.expectStatus == http.StatusOK {
				var resp map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectQueueVal, resp["queueCode"])
			}
		})
	}
}

func TestRecurringScheduledJobRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectStatus   int
		expectQueueVal string
	}{
		{
			name: "valid with queueCode",
			payload: map[string]interface{}{
				"queueCode":  "cron-queue",
				"vnamespace": "default",
				"every":      "5m",
			},
			expectStatus:   http.StatusOK,
			expectQueueVal: "cron-queue",
		},
		{
			name: "valid with targetCode fallback",
			payload: map[string]interface{}{
				"targetCode": "cron-queue-fallback",
				"vnamespace": "default",
				"every":      "1h",
			},
			expectStatus:   http.StatusOK,
			expectQueueVal: "cron-queue-fallback",
		},
		{
			name: "missing both returns 400",
			payload: map[string]interface{}{
				"vnamespace": "default",
				"every":      "5m",
			},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/test-recurring", func(c *gin.Context) {
				var req createRecurringScheduledJobRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
					return
				}

				queueCode := resolveQueueCode(req.QueueCode, req.TargetCode)
				if queueCode == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "queueCode is required"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"queueCode": queueCode})
			})

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/test-recurring", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.expectStatus == http.StatusOK {
				var resp map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectQueueVal, resp["queueCode"])
			}
		})
	}
}
