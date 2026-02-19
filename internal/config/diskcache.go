package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func diskcacheValidate(ydc *yamlDiskCache) (*DiskCache, error) {
	if ydc.Directory == "" {
		return nil, fmt.Errorf("disable disk-cache, because %s is empty value", ydc.Directory)
	}

	if ydc.Size.Size() < 10*1000*1000 {
		return nil, fmt.Errorf("disable disk-cache, because size is small (>10MB)")
	}

	root, err := os.OpenRoot(ydc.Directory)
	if err != nil {
		return nil, fmt.Errorf("disable disk-cache: %s", err.Error())
	}
	defer root.Close() // nolint

	// 空ディレクトリであることを確認する
	dot, err := os.Open(ydc.Directory)
	if err != nil {
		return nil, fmt.Errorf("disable disk-cache: %s", err.Error())
	}

	// ディレクトリが空であれば、エントリ数が0になり、io.EOFを返す
	entry, err := dot.ReadDir(1)
	if len(entry) > 0 || !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("disable disk-cache: %s is not empty directory", ydc.Directory)
	}

	// ファイルの読み書き試験
	var (
		dirname  = "testdir"
		filename = filepath.Join(dirname, "testfile")

		dirPerm os.FileMode = 0755
	)
	if err = root.Mkdir(dirname, dirPerm); err != nil {
		return nil, fmt.Errorf("disable disk-cache: %s", err.Error())
	}
	if err = root.WriteFile(filename, []byte("test"), dirPerm); err != nil {
		return nil, fmt.Errorf("disable disk-cache: %s", err.Error())
	}
	if err = root.RemoveAll(dirname); err != nil {
		return nil, fmt.Errorf("disable disk-cache: %s", err.Error())
	}

	return &DiskCache{
		Directory: ydc.Directory,
		Size:      ydc.Size,
	}, nil
}
