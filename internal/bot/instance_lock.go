package bot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

var ErrAlreadyRunning = errors.New("another bot instance is already running")

type InstanceLock struct {
	file *os.File
	path string
}

func AcquireInstanceLock(dataDir string) (*InstanceLock, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(dataDir, "bot.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyRunning, path)
		}
		return nil, err
	}

	if err := file.Truncate(0); err != nil {
		file.Close()
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		file.Close()
		return nil, err
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		file.Close()
		return nil, err
	}

	return &InstanceLock{
		file: file,
		path: path,
	}, nil
}

func (l *InstanceLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		l.file.Close()
		l.file = nil
		return err
	}
	err := l.file.Close()
	l.file = nil
	return err
}
