// Copyright 2018 The LevelDB-Go and Pebble Authors. All rights reserved. Use
// of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

package pebble

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/petermattis/pebble/db"
	"github.com/petermattis/pebble/storage"
)

type syncedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncedBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestEventListener(t *testing.T) {
	var buf syncedBuffer

	d, err := Open("", &db.Options{
		Storage: storage.NewMem(),
		EventListener: &db.EventListener{
			CompactionBegin: func(info db.CompactionInfo) {
				fmt.Fprintf(&buf, "#%d: compaction begin: L%d -> L%d\n", info.JobID,
					info.Input.Level, info.Input.Level+1)
			},
			CompactionEnd: func(info db.CompactionInfo) {
				fmt.Fprintf(&buf, "#%d: compaction end: L%d -> L%d\n", info.JobID,
					info.Input.Level, info.Input.Level+1)
			},
			FlushBegin: func(info db.FlushInfo) {
				fmt.Fprintf(&buf, "#%d: flush begin\n", info.JobID)
			},
			FlushEnd: func(info db.FlushInfo) {
				fmt.Fprintf(&buf, "#%d: flush end: %d\n", info.JobID, info.Output.FileNum)
			},
			TableDeleted: func(info db.TableDeleteInfo) {
				fmt.Fprintf(&buf, "#%d: table deleted: %d\n", info.JobID, info.FileNum)
			},
			TableIngested: func(info db.TableIngestInfo) {
				fmt.Fprintf(&buf, "#%d: table ingested\n", info.JobID)
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := d.Set([]byte("a"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := d.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := d.Compact([]byte("a"), []byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := d.Delete([]byte("a"), nil); err != nil {
		t.Fatal(err)
	}
	if err := d.Compact([]byte("a"), []byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	expected := `#2: flush begin
#2: flush end: 6
#3: compaction begin: L0 -> L1
#3: compaction end: L0 -> L1
#4: flush begin
#4: flush end: 8
#5: compaction begin: L0 -> L1
#5: compaction end: L0 -> L1
#5: table deleted: 6
#5: table deleted: 8
`
	if v := buf.String(); expected != v {
		t.Fatalf("expected\n%s\nbut found\n%s", expected, v)
	}
}
