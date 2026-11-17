package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	_ "golang.org/x/image/tiff" // 让 ui 的解码器认识 TIFF(image.Decode 注册表是全局的)
)

// @author ixxxxoooo

// 图片解码:mygo 自带 PNG / JPEG / GIF(首帧)/ WebP / BMP / SVG,这里补上 TIFF,
// 其余格式(HEIC / HEIF / AVIF / ICO / ICNS / PSD / TGA / JPEG 2000 …)在 macOS 上借系统的 sips 转成 PNG 再解码,
// 所以文件查看器与差异标签看到的图片格式范围一致。

var imgLog = diag.L("image")

// imageExts 是按图片查看的扩展名(小写,带点)。
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".jpe": true, ".jfif": true, ".gif": true, ".webp": true, ".bmp": true,
	".tif": true, ".tiff": true, ".svg": true, ".ico": true, ".icns": true, ".heic": true, ".heif": true,
	".avif": true, ".jp2": true, ".psd": true, ".tga": true,
}

// isImageExt 判断扩展名(如 ".PNG")是不是图片。
func isImageExt(ext string) bool { return imageExts[strings.ToLower(ext)] }

// decodeAnyImage 解码任意常见格式的图片。
// 参数:data 为文件原始字节;ext 为扩展名(给 sips 当临时文件后缀,可为空)。
// 返回:可直接交给 ui.Image 的图片源;全部失败时返回错误。
func decodeAnyImage(data []byte, ext string) (ui.ImageSource, error) {
	if len(data) == 0 {
		return nil, errors.New("文件为空")
	}
	img, err := ui.DecodeImage(data)
	if err == nil && img != nil {
		return img, nil
	}
	if runtime.GOOS != "darwin" {
		return nil, err
	}
	png, serr := sipsToPNG(data, ext)
	if serr != nil {
		imgLog.Warn("系统转换图片失败", "ext", ext, "err", serr)
		return nil, err
	}
	return ui.DecodeImage(png)
}

// sipsToPNG 用 macOS 自带的 sips 把图片转成 PNG,返回 PNG 字节。超时 20 秒。
func sipsToPNG(data []byte, ext string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "minicode-img-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if ext == "" || strings.ContainsAny(ext, "/\\ ") {
		ext = ".img"
	}
	in, out := filepath.Join(dir, "in"+ext), filepath.Join(dir, "out.png")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, err
	}
	end := diag.SpanSlow("image", "sips 转换", 800*time.Millisecond, "ext", ext)
	defer end()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "sips", "-s", "format", "png", in, "--out", out)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return os.ReadFile(out)
}

// looksBinary 粗判一段内容是不是二进制(前 8000 字节里含 NUL)。
func looksBinary(s string) bool {
	if len(s) > 8000 {
		s = s[:8000]
	}
	return strings.IndexByte(s, 0) >= 0
}
