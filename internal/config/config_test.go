package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetDefaults(t *testing.T) {
	cfg := Config{}
	setDefaults(&cfg)
	if cfg.ReconnectInterval != 1 || cfg.ReconnectMaxInterval != 30 || cfg.ReconnectJitterMs != 0 {
		t.Fatalf("reconnect defaults not applied: interval=%d max=%d jitter=%d",
			cfg.ReconnectInterval, cfg.ReconnectMaxInterval, cfg.ReconnectJitterMs)
	}
	if cfg.UpstreamQueueSize != 8192 || cfg.UpstreamBatchMaxPackets != 256 || cfg.UpstreamBatchMaxBytes != 4*1024*1024 {
		t.Fatalf("upstream batch defaults not applied: queue=%d packets=%d bytes=%d",
			cfg.UpstreamQueueSize, cfg.UpstreamBatchMaxPackets, cfg.UpstreamBatchMaxBytes)
	}
	if cfg.UpstreamLaneCount != 4 || cfg.DownstreamLaneCount != 4 {
		t.Fatalf("lane defaults not applied: up=%d down=%d", cfg.UpstreamLaneCount, cfg.DownstreamLaneCount)
	}
	if cfg.UpstreamReorderWindow != 128 || cfg.UpstreamReorderHoldMs != 3 {
		t.Fatalf("upstream reorder defaults not applied: window=%d hold=%d",
			cfg.UpstreamReorderWindow, cfg.UpstreamReorderHoldMs)
	}
	if cfg.DownstreamBatchMaxPackets != 128 || cfg.DownstreamBatchMaxBytes != 512*1024 || cfg.DownstreamBatchMaxDelayMs != 1 {
		t.Fatalf("downstream batch defaults not applied: packets=%d bytes=%d delay=%d",
			cfg.DownstreamBatchMaxPackets, cfg.DownstreamBatchMaxBytes, cfg.DownstreamBatchMaxDelayMs)
	}
	if cfg.HeartbeatIntervalMs != 1000 {
		t.Fatalf("heartbeat default not applied: %d", cfg.HeartbeatIntervalMs)
	}
	if cfg.UpstreamCrypto.Type != "none" || cfg.DownstreamCrypto.Type != "none" {
		t.Fatalf("crypto defaults not applied: %+v %+v", cfg.UpstreamCrypto, cfg.DownstreamCrypto)
	}
}

func TestLoadIgnoresTuningFieldsFromJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	data := []byte(`{
  "mode":"client",
  "listen_addr":"0.0.0.0:51820",
  "server_upstream":"127.0.0.1:9001",
  "server_downstream":"127.0.0.1:9002",
  "upstream_queue_size":1,
  "upstream_batch_max_packets":1,
  "upstream_batch_max_bytes":1,
  "upstream_batch_max_delay_ms":99,
  "upstream_lane_count":99,
  "downstream_lane_count":99,
  "downstream_batch_max_packets":1,
  "downstream_batch_max_bytes":1,
  "downstream_batch_max_delay_ms":99,
  "heartbeat_interval_ms":99,
  "upstream_stripe_enabled":true,
  "upstream_reorder_window":1,
  "upstream_reorder_hold_ms":1
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.UpstreamQueueSize != 8192 || cfg.UpstreamBatchMaxPackets != 256 || cfg.UpstreamBatchMaxBytes != 4*1024*1024 {
		t.Fatalf("fixed upstream tuning not applied: %+v", cfg)
	}
	if cfg.UpstreamLaneCount != 4 || cfg.DownstreamLaneCount != 4 {
		t.Fatalf("fixed lane counts not applied: up=%d down=%d", cfg.UpstreamLaneCount, cfg.DownstreamLaneCount)
	}
	if cfg.DownstreamBatchMaxPackets != 128 || cfg.DownstreamBatchMaxBytes != 512*1024 || cfg.DownstreamBatchMaxDelayMs != 1 {
		t.Fatalf("fixed downstream tuning not applied: packets=%d bytes=%d delay=%d",
			cfg.DownstreamBatchMaxPackets, cfg.DownstreamBatchMaxBytes, cfg.DownstreamBatchMaxDelayMs)
	}
	if cfg.HeartbeatIntervalMs != 1000 || cfg.UpstreamStripeEnabled {
		t.Fatalf("fixed control tuning not applied: heartbeat=%d stripe=%v", cfg.HeartbeatIntervalMs, cfg.UpstreamStripeEnabled)
	}
}
