// Copyright 2024 Sudo Sweden AB
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

package templates

import (
	"embed"
	"html/template"
)

//go:embed *.tmpl
var templateFS embed.FS

var Tmpl = template.Must(parse())

func parse() (*template.Template, error) {
	// Parse all .tmpl files inside the embed FS
	t, err := template.ParseFS(templateFS, "*.tmpl")
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Get returns a named template
func Get(name string) *template.Template {
	return Tmpl.Lookup(name)
}
