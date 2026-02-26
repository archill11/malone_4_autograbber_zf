package tg_service

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	fontPathRoboto = "./font/Roboto.ttf"
)

// processImageFile открывает изображение с диска, применяет uniqueImage и сохраняет результат.
func UniqueProcessImageFile(inputPath, outputPath string) error {
	src, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}
	out := uniqueImage(src)
	// imaging.Save выбирает формат по расширению outputPath.
	if err := imaging.Save(out, outputPath); err != nil {
		return err
	}
	return nil
}

// processVideoFile запускает uniqueVideo для указанных файлов.
func UniqueProcessVideoFile(inputPath, outputPath string, is_krug bool) error {
	return uniqueVideo(inputPath, outputPath, is_krug)
}

// uniqueImage applies random geometric and color transforms and optionally overlays a timestamp text.
func uniqueImage(src image.Image) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxCropX := minInt(40, width/10)
	maxCropY := minInt(40, height/10)

	left, rightMargin, top, bottomMargin := 0, 0, 0, 0
	if maxCropX > 0 {
		left = rand.Intn(maxCropX + 1)
		rightMargin = rand.Intn(maxCropX + 1)
	}
	if maxCropY > 0 {
		top = rand.Intn(maxCropY + 1)
		bottomMargin = rand.Intn(maxCropY + 1)
	}

	rect := image.Rect(
		bounds.Min.X+left,
		bounds.Min.Y+top,
		bounds.Max.X-rightMargin,
		bounds.Max.Y-bottomMargin,
	)
	if rect.Dx() > 0 && rect.Dy() > 0 {
		src = imaging.Crop(src, rect)
		bounds = src.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	// Небольшой случайный поворот без флипов, чтобы не портить текст.
	if rand.Float64() < 0.7 { // 70% картинок слегка поворачиваем
		angle := (rand.Float64()*2 - 1) // -1..1 градус
		src = imaging.Rotate(src, angle, color.Transparent)
	}

	src = imaging.AdjustBrightness(src, rand.Float64()*20-10) // -10..10
	src = imaging.AdjustContrast(src, rand.Float64()*20-10)   // -10..10

	// Try to overlay a small timestamp text using either UNIQ_FONT_PATH or a common system font.
	fontPath := fontPathRoboto
	if fontPath != "" {
		dc := gg.NewContextForImage(src)
		if err := dc.LoadFontFace(fontPath, 12); err == nil {
			dc.SetRGBA(1, 1, 1, 0.5)

			x := float64(width/10 + rand.Intn(maxInt(1, width/2)))
			y := float64(height/10 + rand.Intn(maxInt(1, height/2)))
			dc.DrawStringAnchored(fmt.Sprintf("%v_%v", rand.Intn(999), time.Now().Format("15:04:05.000")), x, y, 0, 0)

			return dc.Image()
		}
	}

	return src
}

// uniqueVideo applies crop, scale, brightness/contrast, optional flip, and text overlay (like image pipeline).
func uniqueVideo(inPath, outPath string, is_krug bool) error {
	// Random crop from edges (pixels). Keep modest so we don't lose too much.
	left := rand.Intn(21)       // 0..20
	top := rand.Intn(21)       // 0..20
	right := rand.Intn(21)     // 0..20
	bottom := rand.Intn(21)    // 0..20
	cropW := fmt.Sprintf("trunc((iw-%d-%d)/2)*2", left, right)
	cropH := fmt.Sprintf("trunc((ih-%d-%d)/2)*2", top, bottom)
	cropFilter := fmt.Sprintf("crop=%s:%s:%d:%d", cropW, cropH, left, top)

	// Scale to ensure even dimensions for libx264 (crop may leave odd).
	scaleFilter := "scale=trunc(iw/2)*2:trunc(ih/2)*2"

	// Brightness/contrast — усиленно, чтобы было заметно как на фото (±10% там = здесь примерно ±0.15..0.2 и 0.85..1.15).
	brightness := 0.2 * (rand.Float64()*2 - 1)   // -0.2..0.2 (заметно)
	contrast := 1.0 + (rand.Float64()*0.3 - 0.15) // 0.85..1.15 (заметно)
	eqFilter := fmt.Sprintf("eq=brightness=%.3f:contrast=%.3f", brightness, contrast)

	// Build filter chain: crop -> scale -> eq -> drawtext(optional)
	vfParts := []string{cropFilter, scaleFilter, eqFilter}

	fontPath := fontPathRoboto
	if is_krug {
		fontPath = ""
	}
	// Текст только из цифр и подчёркивания, без двоеточий/точек, чтобы drawtext не ломал фильтр.
	textLabel := fmt.Sprintf("%03d_%d", rand.Intn(1000), time.Now().Unix())
	if fontPath != "" {
		// drawtext: escape path for ffmpeg (Windows drive colon, backslashes).
		escapedFont := escapeDrawtextPath(fontPath)
		// Random position as % of width/height so it works for any resolution.
		xPct := 5 + rand.Intn(40) // 5..44% from left
		yPct := 5 + rand.Intn(40) // 5..44% from top
		drawFilter := fmt.Sprintf(
			"drawtext=fontfile='%s':text='%s':x=w*%d/100:y=h*%d/100:fontsize=22:fontcolor=white@0.5",
			escapedFont, textLabel, xPct, yPct,
		)
		vfParts = append(vfParts, drawFilter)
	}

	vf := strings.Join(vfParts, ",")

	cmd := ffmpeg.Input(inPath).
		Output(outPath, ffmpeg.KwArgs{
			"c:v":   "libx264",
			"preset": "veryfast",
			"vf":    vf,
		}).
		OverWriteOutput().
		ErrorToStdOut()

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// escapeDrawtextPath подготавливает путь к шрифту для fontfile в drawtext.
// Здесь достаточно:
// - привести путь к виду с прямыми слэшами (C:/...),
// - экранировать одинарные кавычки, если вдруг встретятся.
func escapeDrawtextPath(path string) string {
	s := filepath.ToSlash(filepath.Clean(path))
	// Экранируем одинарные кавычки внутри строки, которая будет в '...'.
	s = strings.ReplaceAll(s, "'", `\'`)
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}