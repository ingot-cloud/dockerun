package handler

import (
	"dockerun/internal/config"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type TemplateFile struct {
	FileName    string `json:"file_name"`
	FilePath    string `json:"file_path"`
	Source      string `json:"source"`
	FileType    string `json:"file_type"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

type TemplateMeta struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Icon        string            `json:"icon"`
	MainScript  string            `json:"main_script"`
	WorkDir     string            `json:"work_dir"`
	EnvVars     map[string]string `json:"env_vars"`
	Files       []TemplateFile    `json:"files"`
}

type ScriptTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Icon        string            `json:"icon"`
	MainScript  string            `json:"main_script"`
	WorkDir     string            `json:"work_dir"`
	EnvVars     map[string]string `json:"env_vars"`
	Files       []TemplateFile    `json:"files"`
}

// getTemplateDir 获取模板目录路径
func getTemplateDir() string {
	cfg := config.Load()
	return cfg.TemplateDir
}

// loadTemplatesFromFS 从文件系统加载所有模板
func loadTemplatesFromFS() ([]ScriptTemplate, error) {
	templates := []ScriptTemplate{}

	// 1. 从配置的模板目录加载
	templateDir := getTemplateDir()
	log.Printf("Loading templates from: %s\n", templateDir)

	// 检查模板目录是否存在
	if _, err := os.Stat(templateDir); !os.IsNotExist(err) {
		// 读取模板目录
		entries, err := os.ReadDir(templateDir)
		if err != nil {
			return nil, err
		}

		log.Println("entries: ", entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			templatePath := filepath.Join(templateDir, entry.Name())
			// 新版应用模板可只有 template.json，不作为损坏的历史脚本模板处理。
			if _, err := os.Stat(filepath.Join(templatePath, "meta.json")); os.IsNotExist(err) {
				continue
			}
			template, err := loadTemplate(templatePath)
			if err != nil {
				// 跳过无效的模板，记录错误
				fmt.Printf("Failed to load template %s: %v\n", entry.Name(), err)
				continue
			}

			templates = append(templates, template)
		}
	} else {
		log.Printf("Template directory not found: %s, error: %v\n", templateDir, err)
	}

	// 2. 检查当前工作目录是否包含 meta.json
	currentDir, err := os.Getwd()
	if err == nil {
		metaPath := filepath.Join(currentDir, "meta.json")
		if _, err := os.Stat(metaPath); err == nil {
			log.Printf("Found meta.json in current directory: %s\n", currentDir)
			template, err := loadTemplate(currentDir)
			if err != nil {
				log.Printf("Failed to load template from current directory: %v\n", err)
			} else {
				// 标记为来自当前目录
				template.ID = "current-" + template.ID
				template.Category = "Current Directory"
				templates = append(templates, template)
				log.Printf("Loaded template from current directory: %s\n", template.Name)
			}
		}
	}

	return templates, nil
}

// loadTemplate 加载单个模板
func loadTemplate(templatePath string) (ScriptTemplate, error) {
	var template ScriptTemplate

	// 读取 meta.json
	metaPath := filepath.Join(templatePath, "meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return template, fmt.Errorf("failed to read meta.json: %w", err)
	}

	var meta TemplateMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return template, fmt.Errorf("failed to parse meta.json: %w", err)
	}

	// 读取主脚本
	mainScriptPath := filepath.Join(templatePath, meta.MainScript)
	mainScriptContent, err := os.ReadFile(mainScriptPath)
	if err != nil {
		return template, fmt.Errorf("failed to read main script: %w", err)
	}

	// 构建模板对象
	template = ScriptTemplate{
		ID:          meta.ID,
		Name:        meta.Name,
		Description: meta.Description,
		Category:    meta.Category,
		Icon:        meta.Icon,
		MainScript:  string(mainScriptContent),
		WorkDir:     meta.WorkDir,
		EnvVars:     meta.EnvVars,
		Files:       make([]TemplateFile, 0),
	}

	// 读取附加文件
	for _, fileInfo := range meta.Files {
		if fileInfo.Source == "" {
			continue
		}

		filePath := filepath.Join(templatePath, fileInfo.Source)
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", fileInfo.Source, err)
			continue
		}

		template.Files = append(template.Files, TemplateFile{
			FileName:    fileInfo.FileName,
			FilePath:    fileInfo.FilePath,
			Source:      fileInfo.Source,
			FileType:    fileInfo.FileType,
			Content:     string(content),
			Description: fileInfo.Description,
		})
	}

	return template, nil
}

// GetTemplates 获取所有模板
func GetTemplates(c *gin.Context) {
	templates, err := loadTemplatesFromFS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetTemplate 获取单个模板
func GetTemplate(c *gin.Context) {
	id := c.Param("id")

	templates, err := loadTemplatesFromFS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, tmpl := range templates {
		if tmpl.ID == id {
			c.JSON(http.StatusOK, tmpl)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
}
