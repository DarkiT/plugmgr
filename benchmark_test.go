package plugmgr

import (
	"path/filepath"
	"testing"
)

func BenchmarkLoadPlugin(b *testing.B) {
	path := filepath.Join("testdata", "sample_plugin.so")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadPlugin(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecutePlugin(b *testing.B) {
	m, err := NewManager("./testdata", "config.json")
	if err != nil {
		b.Fatal(err)
	}

	if err := m.LoadPlugin(filepath.Join("testdata", "sample_plugin.so")); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.ExecutePlugin("sample_plugin", "test data")
		if err != nil {
			b.Fatal(err)
		}
	}
}
