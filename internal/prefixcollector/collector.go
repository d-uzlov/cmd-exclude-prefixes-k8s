// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package prefixcollector contains excluded prefix collector, prefix sources and functions working with them
package prefixcollector

import (
	"cmd-exclude-prefixes-k8s/internal/utils"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
)

// ExcludePrefixSource is source of excluded prefixes
type ExcludePrefixSource interface {
	Prefixes() []string
}

// ExcludePrefixCollector is service, collecting excluded prefixes from list of ExcludePrefixSource
// and environment variable "EXCLUDED_PREFIXES"
// and writing result to outputFilePath in yaml format
type ExcludePrefixCollector struct {
	notify         *sync.Cond
	outputFilePath string
	sources        []ExcludePrefixSource
}

const (
	// DefaultConfigMapName is default name of nsm kubernetes ConfigMap
	DefaultConfigMapName  = "nsm-config-volume"
	outputFilePermissions = 0600
)

// NewExcludePrefixCollector creates ExcludePrefixCollector
func NewExcludePrefixCollector(sources []ExcludePrefixSource, outputFilePath string, notify *sync.Cond) *ExcludePrefixCollector {
	collector := &ExcludePrefixCollector{
		outputFilePath: outputFilePath,
		notify:         notify,
		sources:        sources,
	}

	return collector
}

// Start - starts every source, then begin monitoring notifyChan.
// Updates exclude prefix file after every notification.
func (epc *ExcludePrefixCollector) Start() {
	for {
		epc.notify.L.Lock()
		epc.notify.Wait()
		epc.notify.L.Unlock()
		epc.updateExcludedPrefixesConfigmap()
	}
}

func (epc *ExcludePrefixCollector) updateExcludedPrefixesConfigmap() {
	excludePrefixPool, _ := prefixpool.New()

	for _, v := range epc.sources {
		sourcePrefixes := v.Prefixes()
		if len(sourcePrefixes) == 0 {
			continue
		}

		if err := excludePrefixPool.ReleaseExcludedPrefixes(v.Prefixes()); err != nil {
			logrus.Error(err)
			return
		}
	}

	data, err := utils.PrefixesToYaml(excludePrefixPool.GetPrefixes())
	if err != nil {
		logrus.Errorf("Can not create marshal prefixes, err: %v", err.Error())
		return
	}

	err = ioutil.WriteFile(epc.outputFilePath, data, outputFilePermissions)
	if err != nil {
		logrus.Fatalf("Unable to write into file: %v", err.Error())
	}
}
