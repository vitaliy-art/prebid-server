package adcamp

import (
	"encoding/json"
)

// For use with extra_info
type config struct {
	Token string `json:"token"`
}

func parseConfig(s string) (*config, error) {
	c := &config{}
	err := json.Unmarshal([]byte(s), c)
	return c, err
}
