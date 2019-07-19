package fs

import "github.com/Soontao/hanafs/hana"

// CreateFileSizeProvider func
func CreateFileSizeProvider(client *hana.Client) FileSizeProvider {
	return func(path string) int64 {
		if content, err := client.ReadFile(path); err == nil {
			return int64(len(content))
		}
		return 0
	}
}
