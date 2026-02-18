package diskcache

import (
	"compress/gzip"
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
	"github.com/mackerelio-labs/sabatrafficd/internal/sendqueue"
)

type DiskCache struct {
	mu sync.Mutex

	root  *os.Root
	queue queue
	conf  *config.DiskCache

	filelist   *list.List
	totalBytes int64
	totalItems int

	fileMu    sync.Mutex
	filequeue *list.List
}

type cacheEntry struct {
	filename string
	bytes    int64
	items    int
}

type queue interface {
	FrontN(length int) []sendqueue.Item
	Len() int
}

const limit = 1000

func New(q queue, conf *config.DiskCache) (*DiskCache, error) {
	if conf == nil {
		return nil, fmt.Errorf("non configuration")
	}
	root, err := os.OpenRoot(conf.Directory)
	if err != nil {
		return nil, fmt.Errorf("disable disk-cache: %s", err.Error())
	}

	return &DiskCache{
		root:  root,
		queue: q,
		conf:  conf,

		filelist:  list.New(),
		filequeue: list.New(),
	}, nil
}

func (dc *DiskCache) Close() error {
	return dc.root.Close()
}

func (dc *DiskCache) Tick(ctx context.Context) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.createFile()
	dc.purge()
}

func (dc *DiskCache) createFile() {
	// Tick 環境下の制約により、ファイル名は衝突しない
	filename := fmt.Sprintf("%d.dat.gz", time.Now().UnixMilli())

	// limit 件未満は処理しない
	if len := dc.queue.Len(); len < limit {
		return
	}

	items := dc.queue.FrontN(limit)
	// 詰まりが解消し、0件の取得になった場合はスキップ
	length := len(items)
	if length == 0 {
		return
	}

	fi, err := dc.root.Create(filename)
	if err != nil {
		slog.Error("failed save diskcache", slog.String("error", err.Error()))
		return
	}

	wr := gzip.NewWriter(fi)
	if err = json.NewEncoder(wr).Encode(items); err != nil {
		slog.Error("failed save diskcache", slog.String("error", err.Error()))
		return
	}
	if err = wr.Close(); err != nil {
		slog.Error("failed save diskcache", slog.String("error", err.Error()))
		return
	}
	if err = fi.Close(); err != nil {
		slog.Error("failed save diskcache", slog.String("error", err.Error()))
		return
	}

	var bs int64
	st, err := dc.root.Stat(filename)
	if err == nil {
		bs = st.Size()
	}

	dc.filelist.PushBack(cacheEntry{filename: filename, bytes: bs, items: length})
	dc.totalBytes += bs
	dc.totalItems += length
}

func (dc *DiskCache) purge() {
	if dc.totalBytes < dc.conf.Size.Size() {
		return
	}

	e := dc.filelist.Front()
	if e == nil {
		slog.Error("something wrong", slog.Int64("size", dc.totalBytes))
		return
	}
	dc.filelist.Remove(e)
	entry := e.Value.(cacheEntry)
	if err := dc.root.Remove(entry.filename); err != nil {
		slog.Error("failed remove diskcache", slog.String("filename", entry.filename), slog.String("error", err.Error()))
	} else {
		slog.Info("remove diskcache because disk size limit.", slog.String("filename", entry.filename))
	}
	dc.totalBytes -= entry.bytes
	dc.totalItems -= entry.items
}

func (*DiskCache) Reload(conf *config.CollectorConfig) {
	// no support
}

func (*DiskCache) CollectorID() string {
	// no support
	return ""
}

func (dc *DiskCache) filelistDequeue() (cacheEntry, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	e := dc.filelist.Front()
	if e == nil {
		return cacheEntry{}, false
	}
	dc.filelist.Remove(e)

	return e.Value.(cacheEntry), true
}

func (dc *DiskCache) Dequeue() (hostid string, metrics []*mackerel.MetricValue, ok bool) {
	dc.fileMu.Lock()
	defer dc.fileMu.Unlock()

	// ファイルの読み込みが開始されてない。または、ファイルを読み切った場合
	if dc.filequeue.Len() == 0 {
		// 読み出すべきファイルがあれば、処理する
		if entry, ok := dc.filelistDequeue(); ok {
			fi, err := dc.root.Open(entry.filename)
			if err != nil {
				slog.Error("failed load diskcache", slog.String("error", err.Error()))
				return "", nil, false
			}
			rd, err := gzip.NewReader(fi)
			if err != nil {
				slog.Error("failed load diskcache", slog.String("error", err.Error()))
				return "", nil, false
			}
			var items []sendqueue.Item
			if err = json.NewDecoder(rd).Decode(&items); err != nil {
				slog.Error("failed load diskcache", slog.String("error", err.Error()))
				return "", nil, false
			}
			// container/list にコピーする
			for idx := range items {
				dc.filequeue.PushBack(sendqueue.Item{HostID: items[idx].HostID, Metrics: items[idx].Metrics})
			}
			if err = rd.Close(); err != nil {
				slog.Error("failed load diskcache", slog.String("error", err.Error()))
			}
			if err = fi.Close(); err != nil {
				slog.Error("failed load diskcache", slog.String("error", err.Error()))
			}
			// ファイルは削除する
			if err = dc.root.Remove(entry.filename); err != nil {
				slog.Error("failed remove diskcache", slog.String("filename", entry.filename), slog.String("error", err.Error()))
			} else {
				dc.totalBytes -= entry.bytes
				dc.totalItems -= entry.items
			}
		}
	}

	e := dc.filequeue.Front()
	if e == nil {
		return "", nil, false
	}

	item := e.Value.(sendqueue.Item)
	dc.filequeue.Remove(e)
	return item.HostID, item.Metrics, true
}

// 未送信件数
// ファイルには、制限件数分のデータが格納されている。
// filelist * 制限件数 + 現在読み込んでいるファイルが元の件数を返す
func (dc *DiskCache) Len() int {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.fileMu.Lock()
	defer dc.fileMu.Unlock()
	return dc.totalItems + dc.filequeue.Len()
}

func (dc *DiskCache) ReEnqueue(hostID string, metrics []*mackerel.MetricValue) {
	dc.fileMu.Lock()
	defer dc.fileMu.Unlock()
	dc.filequeue.PushFront(sendqueue.Item{HostID: hostID, Metrics: metrics})
}
