package sensor

import "net/http"

type Fetcher interface {
	Get(url string) (*http.Response, error)
}
