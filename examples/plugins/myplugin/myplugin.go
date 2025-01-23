package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pm "github.com/darkit/plugmgr"
)

var Plugin *FileManagerPlugin

// FileManagerConfig 插件配置结构
type FileManagerConfig struct {
	// 工作目录路径
	WorkDir string `json:"work_dir"`
	// 允许的文件扩展名
	AllowedExtensions []string `json:"allowed_extensions"`
	// 最大文件大小(MB)
	MaxFileSize int64 `json:"max_file_size"`
}

// FileManagerPlugin 插件结构体
type FileManagerPlugin struct {
	config FileManagerConfig
	stats  struct {
		filesProcessed int
		lastAccess     time.Time
	}
}

// FileOperation 文件操作请求结构
type FileOperation struct {
	Operation string `json:"operation"` // create, read, update, delete
	Filename  string `json:"filename"`
	Content   string `json:"content,omitempty"`
}

// FileResponse 文件操作响应结构
type FileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Metadata 返回插件的元数据
func (p *FileManagerPlugin) Metadata() pm.PluginMetadata {
	return pm.PluginMetadata{
		Name:    "filemanager",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"core": ">=1.0.0",
		},
		GoVersion: "1.22",
		Config:    p.config,
	}
}

// PreLoad 在插件加载前执行初始化检查
func (p *FileManagerPlugin) PreLoad(config []byte) error {
	// 如果配置为空，使用默认配置
	if config == nil {
		p.config = FileManagerConfig{
			WorkDir:           "./files",
			AllowedExtensions: []string{".txt", ".log", ".json"},
			MaxFileSize:       10, // 10MB
		}
	} else {
		// 解析配置
		var cfg FileManagerConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("解析配置失败: %v", err)
		}
		p.config = cfg
	}

	// 确保工作目录存在
	if err := os.MkdirAll(p.config.WorkDir, 0o755); err != nil {
		return fmt.Errorf("创建工作目录失败: %v", err)
	}

	return nil
}

// Init 初始化插件
func (p *FileManagerPlugin) Init() error {
	p.stats.filesProcessed = 0
	p.stats.lastAccess = time.Now()
	fmt.Println("文件管理插件初始化完成")
	return nil
}

// PostLoad 插件加载后的处理
func (p *FileManagerPlugin) PostLoad() error {
	// 扫描工作目录，统计现有文件
	files, err := os.ReadDir(p.config.WorkDir)
	if err != nil {
		return fmt.Errorf("扫描工作目录失败: %v", err)
	}
	p.stats.filesProcessed = len(files)
	fmt.Printf("发现 %d 个现有文件\n", p.stats.filesProcessed)
	return nil
}

// PreUnload 插件卸载前的清理工作
func (p *FileManagerPlugin) PreUnload() error {
	fmt.Printf("插件统计: 处理了 %d 个文件\n", p.stats.filesProcessed)
	return nil
}

// Execute 执行文件操作
func (p *FileManagerPlugin) Execute(data interface{}) (interface{}, error) {
	// 更新访问时间
	p.stats.lastAccess = time.Now()

	// 解析操作请求
	op, ok := data.(FileOperation)
	if !ok {
		return nil, fmt.Errorf("无效的操作请求")
	}

	// 验证文件扩展名
	ext := filepath.Ext(op.Filename)
	validExt := false
	for _, allowed := range p.config.AllowedExtensions {
		if ext == allowed {
			validExt = true
			break
		}
	}
	if !validExt {
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}

	filePath := filepath.Join(p.config.WorkDir, op.Filename)

	switch op.Operation {
	case "create":
		return p.createFile(filePath, op.Content)
	case "read":
		return p.readFile(filePath)
	case "update":
		return p.updateFile(filePath, op.Content)
	case "delete":
		return p.deleteFile(filePath)
	default:
		return nil, fmt.Errorf("未知操作: %s", op.Operation)
	}
}

// ManageConfig 管理插件配置
func (p *FileManagerPlugin) ManageConfig(config []byte) ([]byte, error) {
	if config == nil {
		// 返回当前配置
		return json.Marshal(p.config)
	}

	// 更新配置
	var newConfig FileManagerConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	// 验证新配置
	if newConfig.MaxFileSize <= 0 {
		return nil, fmt.Errorf("无效的最大文件大小")
	}
	if len(newConfig.AllowedExtensions) == 0 {
		return nil, fmt.Errorf("必须指定允许的文件扩展名")
	}

	// 应用新配置
	p.config = newConfig

	// 返回更新后的配置
	return json.Marshal(p.config)
}

// 辅助方法：创建文件
func (p *FileManagerPlugin) createFile(path, content string) (interface{}, error) {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return FileResponse{Success: false, Message: "文件已存在"}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}

	p.stats.filesProcessed++
	return FileResponse{
		Success: true,
		Message: "文件创建成功",
	}, nil
}

// 辅助方法：读取文件
func (p *FileManagerPlugin) readFile(path string) (interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return FileResponse{
		Success: true,
		Message: "文件读取成功",
		Data:    string(content),
	}, nil
}

// 辅助方法：更新文件
func (p *FileManagerPlugin) updateFile(path, content string) (interface{}, error) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}

	return FileResponse{
		Success: true,
		Message: "文件更新成功",
	}, nil
}

// 辅助方法：删除文件
func (p *FileManagerPlugin) deleteFile(path string) (interface{}, error) {
	if err := os.Remove(path); err != nil {
		return nil, err
	}

	p.stats.filesProcessed--
	return FileResponse{
		Success: true,
		Message: "文件删除成功",
	}, nil
}

// Shutdown 关闭插件
func (p *FileManagerPlugin) Shutdown() error {
	// 执行插件关闭时的清理工作
	fmt.Printf("插件关闭，最后统计：处理了 %d 个文件，最后访问时间 %v\n",
		p.stats.filesProcessed,
		p.stats.lastAccess)
	return nil
}

// 插件初始化函数
func init() {
	Plugin = &FileManagerPlugin{}
}
