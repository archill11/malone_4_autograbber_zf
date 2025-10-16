package tg_service

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Eyevinn/mp4ff/mp4"
)

// RandomizeMP4Metadata изменяет метаданные MP4-файла на случайные.
// Если inputPath == outputPath, файл изменяется "на месте".
func RandomizeMP4Metadata(inputPath, outputPath string) error {
	// 1. Открываем и парсим MP4-файл
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()

	parsedMp4, err := mp4.DecodeFile(file)
	if err != nil {
		return fmt.Errorf("не удалось распарсить MP4: %v", err)
	}

	// 2. Генерируем случайные метаданные
	rand.Seed(time.Now().UnixNano())
	randomTime := time.Now().Add(-time.Duration(randomInt(1, 365*24)) * time.Hour)

	// Меняем даты в Movie Header
	parsedMp4.Moov.Mvhd.CreationTime = uint64(randomTime.Unix())
	parsedMp4.Moov.Mvhd.ModificationTime = uint64(randomTime.Unix())

	// Меняем TrackID и даты в треках
	for _, trak := range parsedMp4.Moov.Traks {
		trak.Tkhd.TrackID = uint32(randomInt(1, 1000))
		trak.Tkhd.CreationTime = uint64(randomTime.Unix())
		trak.Tkhd.ModificationTime = uint64(randomTime.Unix())
	}

	// 3. Если inputPath == outputPath, работаем через временный файл
	if inputPath == outputPath {
		tempPath := inputPath + ".tmp"

		// Создаём временный файл
		tempFile, err := os.Create(tempPath)
		if err != nil {
			return fmt.Errorf("не удалось создать временный файл: %v", err)
		}

		// Записываем изменённые данные во временный файл
		err = parsedMp4.Encode(tempFile)
		tempFile.Close()
		if err != nil {
			return fmt.Errorf("не удалось сохранить временный MP4: %v", err)
		}

		// Заменяем исходный файл временным
		err = os.Rename(tempPath, inputPath)
		if err != nil {
			return fmt.Errorf("не удалось заменить файл: %v", err)
		}

		return nil
	}

	// 4. Если пути разные, просто сохраняем в outputPath
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("не удалось создать выходной файл: %v", err)
	}
	defer outFile.Close()

	err = parsedMp4.Encode(outFile)
	if err != nil {
		return fmt.Errorf("не удалось сохранить MP4: %v", err)
	}

	return nil
}

// randomInt генерирует случайное число в диапазоне [min, max]
func randomInt(min, max int) int {
	n := rand.Intn(max-min+1)
	return min + n
}