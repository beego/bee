// Copyright 2020
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dev

import (
	"os"

	beeLogger "github.com/beego/bee/v2/logger"
)

var preCommit = `
goimports -w -format-only ./ \
ineffassign . \
staticcheck -show-ignored -checks "-ST1017,-U1000,-ST1005,-S1034,-S1012,-SA4006,-SA6005,-SA1019,-SA1024" ./ \
`

// for now, we simply override pre-commit file
func initGitHook() {
	// pcf => pre-commit file
	pcfPath := "./.git/hooks/pre-commit"
	pcf, err := os.OpenFile(pcfPath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		beeLogger.Log.Errorf("try to create or open file failed: %s, cause: %s", pcfPath, err.Error())
		return
	}

	defer pcf.Close()
	_, err = pcf.Write(([]byte)(preCommit))

	if err != nil {
		beeLogger.Log.Errorf("could not init githooks: %s", err.Error())
	} else {
		beeLogger.Log.Successf("The githooks has been added, the content is:\n %s ", preCommit)
	}
}
