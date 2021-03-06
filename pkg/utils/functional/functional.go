/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package f

import (
	"encoding/json"
	"math"

	"github.com/awslabs/karpenter/pkg/utils/log"
)

// GreaterThanInt32 returns values greater than the target value
func GreaterThanInt32(values []int32, target int32) (results []int32) {
	return FilterInt32(values, target, func(a int32, b int32) bool {
		return a > b
	})
}

// LessThanInt32 returns values less than the target value
func LessThanInt32(values []int32, target int32) (results []int32) {
	return FilterInt32(values, target, func(a int32, b int32) bool {
		return a < b
	})
}

// Filter returns values for which the predicate returns true
func FilterInt32(values []int32, target int32, predicate func(a int32, b int32) bool) (results []int32) {
	for _, value := range values {
		if predicate(value, target) {
			results = append(results, value)
		}
	}
	return results
}

// MaxInt32 returns the maximum value in the slice.
func MaxInt32(values []int32) int32 {
	return SelectInt32(values, func(got int32, want int32) int32 {
		return int32(math.Max(float64(got), float64((want))))
	})
}

// MinInt32 returns the minimum value in the slice.
func MinInt32(values []int32) int32 {
	return SelectInt32(values, func(got int32, want int32) int32 {
		return int32(math.Min(float64(got), float64((want))))
	})
}

// SelectInt32 returns the victor of the slice selected by the comparison function.
func SelectInt32(values []int32, selector func(int32, int32) int32) int32 {
	selected := values[0]
	for _, value := range values {
		selected = selector(selected, int32(value))
	}
	return selected
}

/* MergeInto overlays multiple srcs onto a dest struct. Srcs are applied in
order, so srcs[1] will override any fields set from srcs[2]

For example,

dest {a: 1 b: 2}
srcs[0] {a: 2 c: 3}

result {a: 2 b: 2 c: 3}

*/
func MergeInto(dest interface{}, srcs ...interface{}) {
	for _, src := range srcs {
		if src != nil {
			bytes, err := json.Marshal(src)
			log.PanicIfError(err, "Failed to marshall json from %v", src)
			err = json.Unmarshal(bytes, dest)
			log.PanicIfError(err, "Failed to unmarshall json %s into %v", string(bytes), dest)
		}
	}
}

// UnionStringMaps merges all key value pairs into a single map, last write wins.
func UnionStringMaps(maps ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// IntersectStringSlice takes the intersection of all string slices
func IntersectStringSlice(slices ...[]string) []string {
	// count occurrences
	counts := map[string]int{}
	for _, strings := range slices {
		for _, s := range UniqueStrings(strings) {
			counts[s] = counts[s] + 1
		}
	}
	// select if occured in all
	var intersection []string
	for key, count := range counts {
		if count == len(slices) {
			intersection = append(intersection, key)
		}
	}
	return intersection
}

func UniqueStrings(strings []string) []string {
	exists := map[string]bool{}
	for _, s := range strings {
		exists[s] = true
	}
	var unique []string
	for s := range exists {
		unique = append(unique, s)
	}
	return unique
}

// Errorable is a function that returns an error
type Errorable func() error

// AllSucceed returns nil if all errorables return nil, otherwise returns the first error.
func AllSucceed(errorables ...func() error) error {
	for _, errorable := range errorables {
		if err := errorable(); err != nil {
			return err
		}
	}
	return nil
}
