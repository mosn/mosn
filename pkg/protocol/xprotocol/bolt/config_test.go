package bolt

import "testing"

func TestConfigHandler(t *testing.T) {
	t.Run("test bolt config", func(t *testing.T) {
		v := map[string]interface{}{
			"enable_bolt_go_away": true,
		}
		rv := ConfigHandler(v)
		cfg, ok := rv.(Config)
		if !ok {
			t.Fatalf("should returns Config but not")
		}
		if !cfg.EnableBoltGoAway {
			t.Fatalf("invalid config: %v", cfg)
		}
	})
	t.Run("test invalid default", func(t *testing.T) {
		rv := ConfigHandler(nil)
		cfg, ok := rv.(Config)
		if !ok {
			t.Fatalf("should returns Config but not")
		}
		if cfg.EnableBoltGoAway {
			t.Fatalf("invalid config: %v", cfg)
		}
	})
}
