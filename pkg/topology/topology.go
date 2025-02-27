/*
 Copyright 2025, NVIDIA CORPORATION & AFFILIATES

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

package topology

import "gopkg.in/yaml.v2"

type ConfigYaml struct {
	Hosts []Host `yaml:"hosts"`
}

type Host struct {
	Name  string `yaml:"name"`
	Rails []Rail `yaml:"rails"`
}

type Rail struct {
	Tor    string `yaml:"tor"`
	Local  string `yaml:"local"`
	Subnet string `yaml:"subnet"`
}

// config example
// var cfg = `
// hosts:
//   - name: host1
//     rails:
//       - tor: 2.0.0.1
//         local: 2.0.0.2
//   - name: host2
//     rails:
//       - tor: 2.0.0.1
//         local: 2.0.0.3
// `

func LoadConfig(input string) (*ConfigYaml, error) {
	cfg := &ConfigYaml{}

	// unmarshal into an internface map in order to get only what is needed,
	// otherwise unmarshal will fail
	var intermediate map[string]interface{}
	err := yaml.Unmarshal([]byte(input), &intermediate)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(intermediate) // Remarshal cleaned data
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

type Config map[string]Host

func LoadConfigMap(input string) (Config, error) {
	cfgyaml, err := LoadConfig(input)
	if err != nil {
		return nil, err
	}

	config := make(Config)

	for i := range cfgyaml.Hosts {
		config[cfgyaml.Hosts[i].Name] = cfgyaml.Hosts[i]
	}

	return config, nil
}
