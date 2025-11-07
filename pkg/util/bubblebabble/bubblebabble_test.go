// Copyright 2025 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bubblebabble_test

import (
	"testing"

	"github.com/sudoswedenab/dockyards-backend/pkg/util/bubblebabble"
)

func TestBubbleBabble(t *testing.T) {
	t.Run("well known inputs and outputs", func(t *testing.T) {
		cases := []struct{
			input string
			output string
		}{
			{
				input: "",
				output: "xexax",
			},
			{
				input: "1234567890",
				output: "xesef-disof-gytuf-katof-movif-baxux",
			},
			{
				input: "Pineapple",
				output: "xigak-nyryk-humil-bosek-sonax",
			},
		}
		for _, c := range cases {
			output := bubblebabble.BubbleBabble(c.input)
			if output != c.output {
				t.Fatalf("expected %s, but got %s", c.output, output)
			}
		}
	})
}
