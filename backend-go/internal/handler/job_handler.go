package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/repository"
	"adminx/internal/service"
)

// JobHandler 定时任务 HTTP 处理器。
type JobHandler struct {
	audit  *service.AuditService
	svc    *service.JobService
	repo   *repository.JobRepo
	logger *slog.Logger
}

func NewJobHandler(svc *service.JobService, repo *repository.JobRepo, audit *service.AuditService, logger *slog.Logger) *JobHandler {
	return &JobHandler{svc: svc, repo: repo, audit: audit, logger: logger}
}

func (h *JobHandler) List(c *gin.Context) {
	p := pagination.Parse(c)
	jobs, count, err := h.svc.List(p.Offset(), p.Size, c.Query("search"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, jobs)
}

func (h *JobHandler) Get(c *gin.Context) {
	job, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, job)
}

func (h *JobHandler) Create(c *gin.Context) {
	var in service.JobCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	job, err := h.svc.Create(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "create", "Job", job.ID, job.Name, nil, job)
	response.Created(c, job)
}

func (h *JobHandler) Update(c *gin.Context) {
	var in service.JobUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	old, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	job, err := h.svc.Update(c.Param("id"), in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "update", "Job", job.ID, job.Name, old, job)
	response.OK(c, job)
}

func (h *JobHandler) Delete(c *gin.Context) {
	old, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	if err := h.svc.Delete(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "delete", "Job", c.Param("id"), old.Name, old, nil)
	response.NoContent(c)
}

// RunOnce POST /jobs/:id/run_once/ — 异步执行，立即返回 logID（调用方轮询 logs 拿结果）。
func (h *JobHandler) RunOnce(c *gin.Context) {
	job, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	logID, err := h.svc.RunOnce(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditOperation(c, h.audit, "execute", "Job", job.ID, job.Name, "手动执行任务", job, gin.H{"log_id": logID, "status": "submitted"})
	response.OK(c, gin.H{"log_id": logID})
}

// Status GET /jobs/status/
func (h *JobHandler) Status(c *gin.Context) {
	alive, err := h.repo.IsSchedulerAlive()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, gin.H{
		"scheduler_alive": alive,
		"status":          map[bool]string{true: "online", false: "offline"}[alive],
	})
}

// Reload POST /jobs/reload/ — 手动触发调度器重载。
func (h *JobHandler) Reload(c *gin.Context) {
	h.svc.Reload()
	auditOperation(c, h.audit, "execute", "Job", "scheduler", "调度器", "提交重载", nil, gin.H{"status": "submitted"})
	alive, err := h.repo.IsSchedulerAlive()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, gin.H{
		"scheduler_alive": alive,
		"message":         "已提交重载",
	})
}

// Logs GET /jobs/logs/
func (h *JobHandler) Logs(c *gin.Context) {
	p := pagination.Parse(c)
	logs, count, err := h.svc.ListLogs(p.Offset(), p.Size, c.Query("job_id"), c.Query("status"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, logs)
}
