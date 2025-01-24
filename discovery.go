package plugmgr

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh"
)

type PluginRepository struct {
	URL       string
	SSHKey    string
	PublicKey ssh.PublicKey
}

// SetupRemoteRepository 设置远程插件仓库
//
//	参数:
//	- url: 远程仓库的 URL 地址
//	- sshKeyPath: SSH 密钥文件的路径
//	返回:
//	- *PluginRepository: 配置好的插件仓库对象
//	- error: 设置过程中的错误
//	功能:
//	- 读取并解析 SSH 密钥
//	- 创建并返回插件仓库配置对象
func (m *Manager) SetupRemoteRepository(url, sshKeyPath string) (*PluginRepository, error) {
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, wrap(err, "读取 SSH 密钥失败")
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, wrap(err, "解析 SSH 密钥失败")
	}

	return &PluginRepository{
		URL:       url,
		SSHKey:    string(key),
		PublicKey: signer.PublicKey(),
	}, nil
}

// DeployRepository 部署插件仓库
//
//	参数:
//	- repo: 插件仓库配置对象
//	- localPath: 本地部署路径
//	返回:
//	- error: 部署过程中的错误
//	功能:
//	- 下载 redbean 执行文件
//	- 使用 SSH 或本地方式部署仓库
//	- 验证部署结果
func (m *Manager) DeployRepository(repo *PluginRepository, localPath string) error {
	if err := m.downloadRedbean(localPath); err != nil {
		return wrap(err, "下载 redbean 失败")
	}

	cmd := exec.Command(filepath.Join(localPath, "redbean.com"), "-v")
	if repo.URL != "" {
		cmd = exec.Command("ssh", "-i", repo.SSHKey, repo.URL, filepath.Join(localPath, "redbean.com"), "-v")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return wrapf(err, "部署仓库失败: %s", output)
	}

	m.logger.Info("仓库部署成功", "output", string(output))
	return nil
}

// downloadRedbean 下载 redbean 执行文件
//
//	参数:
//	- localPath: 下载文件的本地保存路径
//	返回:
//	- error: 下载过程中的错误
//	功能:
//	- 从官方地址下载最新版本的 redbean
//	- 保存文件到指定路径
//	- 设置适当的文件权限
func (m *Manager) downloadRedbean(localPath string) error {
	resp, err := http.Get("https://redbean.dev/redbean-latest.com")
	if err != nil {
		return wrap(err, "下载 redbean 失败")
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath.Join(localPath, "redbean.com"))
	if err != nil {
		return wrap(err, "创建 redbean 文件失败")
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return wrap(err, "保存 redbean 文件失败")
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(localPath, "redbean.com"), 0o755); err != nil {
			return wrap(err, "设置 redbean 执行权限失败")
		}
	}

	return nil
}

// VerifyPluginSignature 验证插件签名
//
//	参数:
//	- pluginPath: 插件文件路径
//	- publicKeyPath: 公钥文件路径
//	返回:
//	- error: 验证过程中的错误
//	功能:
//	- 读取插件文件和签名文件
//	- 解析公钥并验证签名
//	- 使用 RSA-SHA256 算法进行签名验证
func (m *Manager) VerifyPluginSignature(pluginPath string, publicKeyPath string) error {
	if publicKeyPath == "" {
		m.logger.Warn("未提供公钥路径,跳过插件签名验证", "plugin", pluginPath)
		return nil
	}

	pluginData, err := os.ReadFile(pluginPath)
	if err != nil {
		return wrap(err, "读取插件文件失败")
	}

	signaturePath := pluginPath + ".sig"
	signatureData, err := os.ReadFile(signaturePath)
	if err != nil {
		return wrap(err, "读取签名文件失败")
	}

	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return wrap(err, "读取公钥文件失败")
	}

	block, _ := pem.Decode(publicKeyData)
	if block == nil {
		return newError("解析包含公钥的 PEM 块失败")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return wrap(err, "解析公钥失败")
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return newError("公钥不是 RSA 公钥")
	}

	hashed := sha256.Sum256(pluginData)
	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], signatureData)
	if err != nil {
		return wrap(err, "验证签名失败")
	}

	return nil
}
