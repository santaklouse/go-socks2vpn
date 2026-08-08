package assets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/santaklouse/go-socks2vpn/internal/platform"
)

const (
	wintunURL       = "https://www.wintun.net/builds/wintun-0.14.1.zip"
	wintunSHA256    = "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51"
	maxDownloadSize = 32 << 20
)

var wintunDLLSHA256 = map[string]string{
	"x86":   "d694fa46ab4cfebcb2632d094c7aa97278eef2f8052438621766d863ae98a931",
	"amd64": "e5da8447dc2c320edc0fc52fa01885c103de8c118481f683643cacc3220dafce",
	"arm":   "daad267411ecdc70a0535e274d2c3e9da3d0084bdac7662cb8424dd4a031b4d9",
	"arm64": "f7ba89005544be9d85231a9e0d5f23b2d15b3311667e2dad0debd344918a3f80",
}

type Logger interface {
	Printf(format string, args ...any)
}

type Downloader struct {
	Client    *http.Client
	WintunURL string
	Log       Logger
}

func New(log Logger) *Downloader {
	return &Downloader{
		Client:    &http.Client{Timeout: 2 * time.Minute},
		WintunURL: wintunURL,
		Log:       log,
	}
}

func DefaultCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("не удалось определить каталог кэша: %w", err)
	}
	return filepath.Join(dir, "go-socks2vpn"), nil
}

// EnsureWintun installs only the native Windows TUN driver. The tun2socks
// engine itself is linked into the Go application and is never downloaded.
func (d *Downloader) EnsureWintun(ctx context.Context, info platform.Info, dir string) (string, error) {
	if info.OS != "windows" {
		return "", nil
	}
	arch := map[string]string{"386": "x86", "amd64": "amd64", "arm": "arm", "arm64": "arm64"}[info.Arch]
	if arch == "" {
		return "", fmt.Errorf("Wintun не поддерживает архитектуру %s", info.Arch)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("не удалось создать каталог %s: %w", dir, err)
	}
	destination := filepath.Join(dir, "wintun.dll")
	if regularFile(destination) {
		existing, err := os.ReadFile(destination)
		if err == nil && verifySHA256(existing, wintunDLLSHA256[arch]) == nil {
			d.printf("Используется проверенный Wintun: %s", destination)
			return destination, nil
		}
		d.printf("Кэшированный Wintun повреждён или изменён; файл будет заменён")
	}
	d.printf("Загрузка нативного драйвера Wintun 0.14.1")
	archive, err := d.download(ctx, d.WintunURL)
	if err != nil {
		return "", fmt.Errorf("не удалось загрузить Wintun: %w", err)
	}
	if err := verifySHA256(archive, wintunSHA256); err != nil {
		return "", fmt.Errorf("проверка Wintun не пройдена: %w", err)
	}
	dll, err := extractExact(archive, "wintun/bin/"+arch+"/wintun.dll")
	if err != nil {
		return "", err
	}
	if err := verifySHA256(dll, wintunDLLSHA256[arch]); err != nil {
		return "", fmt.Errorf("проверка Wintun DLL не пройдена: %w", err)
	}
	if err := atomicWrite(destination, dll, 0o644); err != nil {
		return "", err
	}
	d.printf("Wintun установлен: %s", destination)
	return destination, nil
}

func (d *Downloader) download(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-socks2vpn")
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownloadSize {
		return nil, fmt.Errorf("файл превышает допустимый размер %d MiB", maxDownloadSize>>20)
	}
	return data, nil
}

func verifySHA256(data []byte, expected string) error {
	want, err := hex.DecodeString(expected)
	if err != nil || len(want) != sha256.Size {
		return errors.New("некорректный ожидаемый SHA-256")
	}
	got := sha256.Sum256(data)
	if !bytes.Equal(got[:], want) {
		return fmt.Errorf("SHA-256 не совпадает: получено %x, ожидалось %s", got, expected)
	}
	return nil
}

func extractExact(data []byte, name string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("некорректный ZIP-архив: %w", err)
	}
	for _, file := range r.File {
		if filepath.ToSlash(file.Name) != name {
			continue
		}
		if file.UncompressedSize64 > maxDownloadSize {
			return nil, errors.New("распакованный Wintun DLL слишком велик")
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, maxDownloadSize+1))
		closeErr := reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		return content, nil
	}
	return nil, fmt.Errorf("файл %s отсутствует в архиве Wintun", name)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".wintun-*")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("не удалось установить %s: %w", path, err)
	}
	return nil
}

func regularFile(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.Mode().IsRegular() && stat.Size() > 0
}

func (d *Downloader) printf(format string, args ...any) {
	if d.Log != nil {
		d.Log.Printf(format, args...)
	}
}
