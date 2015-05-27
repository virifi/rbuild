package rbuild

import (
	"runtime"
	"strings"
)

type EnvItem struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Prepend bool   `json:"prepend"`
}

func mergeEnv(baseEnv []string, items []EnvItem) []string {
	envMap := envSliceAsMap(baseEnv)
	envSep := ":"
	if runtime.GOOS == "windows" {
		envSep = ";"
	}
	for _, env := range items {
		origEnv, ok := envMap[env.Name]
		if ok {
			if env.Prepend {
				envMap[env.Name] = env.Value + envSep + origEnv
			} else {
				envMap[env.Name] = origEnv + envSep + env.Value
			}
		} else {
			envMap[env.Name] = env.Value
		}
	}
	return envMapToSlice(envMap)
}

func envSliceAsMap(data []string) map[string]string {
	items := make(map[string]string)
	for _, val := range data {
		splits := strings.SplitN(val, "=", 2)
		key := splits[0]
		value := splits[1]
		items[key] = value
	}
	return items
}
func envMapToSlice(envMap map[string]string) []string {
	var env []string
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}
	return env
}
