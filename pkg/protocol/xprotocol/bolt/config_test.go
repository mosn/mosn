/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bolt

import "testing"

func TestConfigHandler(t *testing.T) {
	t.Run("test bolt config", func(t *testing.T) {
		v := map[string]interface{}{
			"enable_bolt_goaway": true,
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
