// 版权所有 (C) 2024 Matt Dunleavy。保留所有权利。
// 本源代码的使用受 LICENSE 文件中的 MIT 许可证约束。

package pluginmanager

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/darkit/slog"
	"golang.org/x/crypto/ssh"
)

type PluginRepository struct {
	URL       string
	SSHKey    string
	PublicKey ssh.PublicKey
}

func (m *Manager) DiscoverPlugins(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".so" {
			pluginName := strings.TrimSuffix(filepath.Base(path), ".so")
			if err := m.LoadPlugin(path); err != nil {
				m.logger.Warn("加载发现的插件失败", slog.String("plugin", pluginName), slog.Any("error", err))
			} else {
				m.logger.Info("发现并加载了插件", slog.String("plugin", pluginName))
			}
		}
		return nil
	})
}

func (m *Manager) SetupRemoteRepository(url, sshKeyPath string) (*PluginRepository, error) {
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("读取 SSH 密钥失败: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("解析 SSH 密钥失败: %w", err)
	}

	return &PluginRepository{
		URL:       url,
		SSHKey:    string(key),
		PublicKey: signer.PublicKey(),
	}, nil
}

func (m *Manager) DeployRepository(repo *PluginRepository, localPath string) error {
	if err := m.downloadRedbean(localPath); err != nil {
		return err
	}

	cmd := exec.Command(filepath.Join(localPath, "redbean.com"), "-v")
	if repo.URL != "" {
		// 通过 SSH 部署
		cmd = exec.Command("ssh", "-i", repo.SSHKey, repo.URL, filepath.Join(localPath, "redbean.com"), "-v")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("部署仓库失败: %w\n输出: %s", err, string(output))
	}

	m.logger.Info("仓库部署成功", slog.String("output", string(output)))
	return nil
}

func (m *Manager) downloadRedbean(localPath string) error {
	resp, err := http.Get("https://redbean.dev/redbean-latest.com")
	if err != nil {
		return fmt.Errorf("下载 redbean 失败: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath.Join(localPath, "redbean.com"))
	if err != nil {
		return fmt.Errorf("创建 redbean 文件失败: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("保存 redbean 文件失败: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(localPath, "redbean.com"), 0o755); err != nil {
			return fmt.Errorf("设置 redbean 执行权限失败: %w", err)
		}
	}

	return nil
}

func (m *Manager) CheckForUpdates(repo *PluginRepository) ([]string, error) {
	// 实现检查仓库更新的逻辑
	// 通常包括向仓库发送 HTTP 请求
	// 并比较已安装插件的版本与可用版本
	return []string{}, nil
}

func (m *Manager) UpdatePlugin(repo *PluginRepository, pluginName string) error {
	// 实现下载和更新特定插件的逻辑
	return nil
}

func (m *Manager) VerifyPluginSignature(pluginPath string, publicKeyPath string) error {
	// 如果没有提供公钥路径,跳过验证
	if publicKeyPath == "" {
		m.logger.Warn("未提供公钥路径,跳过插件签名验证", slog.String("plugin", pluginPath))
		return nil
	}

	// 读取插件文件
	pluginData, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("读取插件文件失败: %w", err)
	}

	// 读取签名文件
	signaturePath := pluginPath + ".sig"
	signatureData, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("读取签名文件失败: %w", err)
	}

	// 读取公钥
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("读取公钥文件失败: %w", err)
	}

	// 解析公钥
	block, _ := pem.Decode(publicKeyData)
	if block == nil {
		return fmt.Errorf("解析包含公钥的 PEM 块失败")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("解析公钥失败: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("公钥不是 RSA 公钥")
	}

	// 验证签名
	hashed := sha256.Sum256(pluginData)
	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], signatureData)
	if err != nil {
		return fmt.Errorf("验证签名失败: %w", err)
	}

	return nil
}
