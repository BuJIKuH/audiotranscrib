package telegram

import (
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

type AudioProcessor interface {
	Process(data []byte, logger *zap.Logger) ([]byte, string, error)
}
type AudioStrategy int

const (
	StrategyDirect AudioStrategy = iota
	StrategyConvert
)

func detectStrategy(mime string) AudioStrategy {
	mime = strings.ToLower(mime)

	switch {
	case strings.Contains(mime, "ogg"):
		return StrategyDirect
	case strings.Contains(mime, "mpeg"):
		return StrategyConvert
	case strings.Contains(mime, "wav"):
		return StrategyConvert
	default:
		return StrategyConvert
	}
}

func ConvertToPCM16k(data []byte, logger *zap.Logger) ([]byte, error) {
	tmpIn, err := os.CreateTemp("", "input_*.wav")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpIn.Name()) // удаляем после выхода
	if _, err := tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, err
	}
	tmpIn.Close()

	tmpOut, err := os.CreateTemp("", "output_*.wav")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpOut.Name())
	tmpOut.Close()

	cmd := exec.Command(
		"ffmpeg",
		"-i", tmpIn.Name(),
		"-ar", "8000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		tmpOut.Name(),
		"-y",
	)
	if err := cmd.Run(); err != nil {
		logger.Error("ffmpeg conversion failed", zap.Error(err))
		return nil, err
	}

	converted, err := os.ReadFile(tmpOut.Name())
	if err != nil {
		return nil, err
	}

	return converted, nil
}
