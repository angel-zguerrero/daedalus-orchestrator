package jobworker

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type JobWorkerController struct {
	Config      *common.ServerConfing
	JobWorkerBO *bo.JobWorkerBO
}

func NewJobWorkerController(Config *common.ServerConfing) *JobWorkerController {
	api := &JobWorkerController{
		Config:      Config,
		JobWorkerBO: bo.NewJobWorkerBO(Config),
	}
	return api
}

// GetJobWorkerHandler handles GET /rest-api/job-workers/:id
func (ctrl *JobWorkerController) GetJobWorkerHandler(c *gin.Context) {
	jobWorkerID := c.Param("id")
	jobWorker, err := ctrl.JobWorkerBO.GetJobWorker(c.Request.Context(), jobWorkerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "JobWorker",
		"result":  jobWorker,
	})
}

// GetJobWorkersHandler handles GET /rest-api/job-workers
func (ctrl *JobWorkerController) GetJobWorkersHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := ctrl.JobWorkerBO.GetJobWorkers(c.Request.Context(), c.Query("q"), c.Query("status"), c.Query("cursor"), page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.JobWorker{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "JobWorker list",
		"result":  findResult,
	})
}
