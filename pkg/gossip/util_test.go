// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package gossip

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/config"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/randutil"
)

func keyFromInt(i int) roachpb.Key {
	return roachpb.Key(strconv.Itoa(i))
}

// addKV adds a random value for the specified key to the system config.
func addKV(rng *rand.Rand, cfg *config.SystemConfig, key int) {
	newKey := keyFromInt(key)
	modified := false
	for _, oldKV := range cfg.Values {
		if !oldKV.Key.Equal(newKey) {
			modified = true
			break
		}
	}
	newKVs := cfg.Values
	if modified {
		newKVs = make([]roachpb.KeyValue, 0, len(cfg.Values))
		for _, oldKV := range cfg.Values {
			if !oldKV.Key.Equal(newKey) {
				newKVs = append(newKVs, oldKV)
			}
		}
	}
	newKVs = append(newKVs, roachpb.KeyValue{
		Key: newKey,
		Value: roachpb.Value{
			RawBytes: randutil.RandBytes(rng, 100),
		},
	})
	cfg.Values = newKVs
}

// assertModified asserts that the specified keys will be considered "modified"
// when passing the new system config through the filter.
func assertModified(
	t *testing.T, df *SystemConfigDeltaFilter, cfg config.SystemConfig, keys ...int,
) {
	t.Helper()
	var modified []int
	df.ForModified(cfg, func(kv roachpb.KeyValue) {
		key, err := strconv.Atoi(string(kv.Key))
		if err != nil {
			t.Fatal(err)
		}
		modified = append(modified, key)
	})
	if !reflect.DeepEqual(modified, keys) {
		t.Errorf("expected keys modified=%v, found %v", keys, modified)
	}
}

// assertSize asserts that the filter has the specified number of kvs stored.
func assertSize(t *testing.T, df SystemConfigDeltaFilter, size int) {
	t.Helper()
	if l := len(df.lastVals); l != size {
		t.Errorf("expected map size=%d, found %d", size, l)
	}
}

func TestSystemConfigDeltaFilter(t *testing.T) {
	defer leaktest.AfterTest(t)()
	rng, _ := randutil.NewPseudoRand()

	df := MakeSystemConfigDeltaFilter(nil)
	cfg := config.SystemConfig{}

	// Add one key.
	addKV(rng, &cfg, 1)
	assertModified(t, &df, cfg, 1)
	assertSize(t, df, 1)

	// Add two keys.
	addKV(rng, &cfg, 2)
	addKV(rng, &cfg, 3)
	assertModified(t, &df, cfg, 2, 3)
	assertSize(t, df, 3)

	// Modify a key.
	addKV(rng, &cfg, 2)
	assertModified(t, &df, cfg, 2)
	assertSize(t, df, 3)

	// Remove the first key.
	cfg.Values = cfg.Values[1:]
	assertModified(t, &df, cfg)
	assertSize(t, df, 2)
}

func TestSystemConfigDeltaFilterWithKeyPrefix(t *testing.T) {
	defer leaktest.AfterTest(t)()
	rng, _ := randutil.NewPseudoRand()

	df := MakeSystemConfigDeltaFilter(keyFromInt(12))
	cfg := config.SystemConfig{}

	// Add one non-matching key.
	addKV(rng, &cfg, 1)
	assertModified(t, &df, cfg)
	assertSize(t, df, 0)

	// Add one matching key.
	addKV(rng, &cfg, 123)
	assertModified(t, &df, cfg, 123)
	assertSize(t, df, 1)

	// Add two keys, one matching, one non-matching.
	addKV(rng, &cfg, 125)
	addKV(rng, &cfg, 135)
	assertModified(t, &df, cfg, 125)
	assertSize(t, df, 2)

	// Modify two keys, one matching, one non-matching.
	addKV(rng, &cfg, 1)
	addKV(rng, &cfg, 123)
	assertModified(t, &df, cfg, 123)
	assertSize(t, df, 2)
}

func BenchmarkSystemConfigDeltaFilter(b *testing.B) {
	df := MakeSystemConfigDeltaFilter(keyFromInt(1))
	rng, _ := randutil.NewPseudoRand()

	// Create two configs.
	var cfg1, cfg2 config.SystemConfig
	for i := 0; i < 1000; i++ {
		key := i + 100000 // +100000 to match filter
		addKV(rng, &cfg1, key)
	}
	for i := 0; i < 200; i++ {
		key := i + 200000 // +200000 to avoid matching filter
		addKV(rng, &cfg1, key)
	}
	// Copy to cfg2 so that most kvs are shared.
	cfg2.Values = append([]roachpb.KeyValue(nil), cfg1.Values...)

	// Make a few modifications to cfg2.
	for i := 0; i < 20; i++ {
		key := i + 1000000 // +1000000 to match filter and first group
		addKV(rng, &cfg2, key)
	}
	for i := 0; i < 20; i++ {
		key := i + 10000 // +10000 to match filter
		addKV(rng, &cfg2, key)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := cfg1
		if i%2 == 1 {
			cfg = cfg2
		}
		df.ForModified(cfg, func(kv roachpb.KeyValue) {
			_ = kv
		})
	}
}