package pluginmanager

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

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

type PluginRepository struct {
	URL       string
	SSHKey    string
	PublicKey ssh.PublicKey
}

// SetupRemoteRepository 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
func (m *Manager) SetupRemoteRepository(url, sshKeyPath string) (*PluginRepository, error) {
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "读取 SSH 密钥失败")
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "解析 SSH 密钥失败")
	}

	return &PluginRepository{
		URL:       url,
		SSHKey:    string(key),
		PublicKey: signer.PublicKey(),
	}, nil
}

// DeployRepository 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
// - 使用 filepath.Join 来构建路径,提高跨平台兼容性
func (m *Manager) DeployRepository(repo *PluginRepository, localPath string) error {
	if err := m.downloadRedbean(localPath); err != nil {
		return errors.Wrap(err, "下载 redbean 失败")
	}

	cmd := exec.Command(filepath.Join(localPath, "redbean.com"), "-v")
	if repo.URL != "" {
		cmd = exec.Command("ssh", "-i", repo.SSHKey, repo.URL, filepath.Join(localPath, "redbean.com"), "-v")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "部署仓库失败: %s", output)
	}

	m.logger.Info("仓库部署成功", "output", string(output))
	return nil
}

// downloadRedbean 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
// - 使用 io.Copy 来简化文件写入
func (m *Manager) downloadRedbean(localPath string) error {
	resp, err := http.Get("https://redbean.dev/redbean-latest.com")
	if err != nil {
		return errors.Wrap(err, "下载 redbean 失败")
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath.Join(localPath, "redbean.com"))
	if err != nil {
		return errors.Wrap(err, "创建 redbean 文件失败")
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "保存 redbean 文件失败")
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(localPath, "redbean.com"), 0o755); err != nil {
			return errors.Wrap(err, "设置 redbean 执行权限失败")
		}
	}

	return nil
}

// VerifyPluginSignature 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
func (m *Manager) VerifyPluginSignature(pluginPath string, publicKeyPath string) error {
	if publicKeyPath == "" {
		m.logger.Warn("未提供公钥路径,跳过插件签名验证", "plugin", pluginPath)
		return nil
	}

	pluginData, err := os.ReadFile(pluginPath)
	if err != nil {
		return errors.Wrap(err, "读取插件文件失败")
	}

	signaturePath := pluginPath + ".sig"
	signatureData, err := os.ReadFile(signaturePath)
	if err != nil {
		return errors.Wrap(err, "读取签名文件失败")
	}

	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return errors.Wrap(err, "读取公钥文件失败")
	}

	block, _ := pem.Decode(publicKeyData)
	if block == nil {
		return errors.New("解析包含公钥的 PEM 块失败")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "解析公钥失败")
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("公钥不是 RSA 公钥")
	}

	hashed := sha256.Sum256(pluginData)
	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], signatureData)
	if err != nil {
		return errors.Wrap(err, "验证签名失败")
	}

	return nil
}
