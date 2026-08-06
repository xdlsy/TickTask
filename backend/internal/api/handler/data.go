package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ticktask/internal/model"
	"ticktask/internal/service"

	"github.com/gin-gonic/gin"
)

const maxImportSize = 50 << 20 // 50MB

type DataHandler struct {
	svc service.DataService
}

func NewDataHandler(svc service.DataService) *DataHandler {
	return &DataHandler{svc: svc}
}

// Export GET /api/data/export → 下载 JSON
func (h *DataHandler) Export(c *gin.Context) {
	env, err := h.svc.Export()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	name := fmt.Sprintf("ticktask-backup-%s.json", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "application/json", raw)
}

// PreviewImport POST /api/data/import/preview (multipart "file")
func (h *DataHandler) PreviewImport(c *gin.Context) {
	env, err := readBackupUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	prev, err := h.svc.PreviewImport(&env.Data, env.SchemaVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prev)
}

// ApplyImport POST /api/data/import/apply (JSON)
func (h *DataHandler) ApplyImport(c *gin.Context) {
	var req model.ApplyImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.svc.ApplyImport(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// readBackupUpload 读 multipart 文件并解析信封,做基础校验。
func readBackupUpload(c *gin.Context) (*model.BackupEnvelope, error) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("文件格式无效:缺少 file")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxImportSize+1))
	if err != nil {
		return nil, fmt.Errorf("文件格式无效:读取失败")
	}
	if len(raw) > maxImportSize {
		return nil, fmt.Errorf("文件过大(>50MB)")
	}
	var env model.BackupEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("文件格式无效")
	}
	if env.App != "ticktask" {
		return nil, fmt.Errorf("不是有效的 TickTask 备份文件")
	}
	return &env, nil
}
