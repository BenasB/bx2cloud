package images

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFlatPuller_ParseImageName(t *testing.T) {
	var imageNameTests = []struct {
		in      string
		outHost string
		outName string
		outRef  string
	}{
		{"my-registry.com/ubuntu:24.04", "my-registry.com", "ubuntu", "24.04"},
		{"my-registry.com/path/ubuntu:24.04", "my-registry.com", "path/ubuntu", "24.04"},
		{"localhost:1234/ubuntu:24.04", "localhost:1234", "ubuntu", "24.04"},
		{"localhost:1234/path/ubuntu:24.04", "localhost:1234", "path/ubuntu", "24.04"},
		{"path/ubuntu:24.04", "registry-1.docker.io", "path/ubuntu", "24.04"},
		{"ubuntu:24.04", "registry-1.docker.io", "library/ubuntu", "24.04"},
		{"ubuntu", "registry-1.docker.io", "library/ubuntu", "latest"},
	}

	puller := &flatPuller{}

	for _, tt := range imageNameTests {
		t.Run(tt.in, func(t *testing.T) {
			resultRef, resultContext := puller.parseImageName(tt.in)
			if resultContext.host != tt.outHost || resultContext.name != tt.outName || resultRef != tt.outRef {
				t.Fatalf(
					"got %q, %q, %q, want %q, %q, %q",
					resultContext.host,
					resultContext.name,
					resultRef,
					tt.outHost,
					tt.outName,
					tt.outRef,
				)
			}
		})
	}
}

func TestFlatPuller_ParseWwwAuthenticate(t *testing.T) {
	var imageNameTests = []struct {
		in  string
		out map[string]string
	}{
		{"Bearer realm=\"https://auth.docker.io/token\",service=\"registry.docker.io\"", map[string]string{
			"realm":   "https://auth.docker.io/token",
			"service": "registry.docker.io",
		}},
		{"Bearer realm=\"https://quay.io/v2/auth\",service=\"quay.io\"", map[string]string{
			"realm":   "https://quay.io/v2/auth",
			"service": "quay.io",
		}},
		{"Bearer custom=my-value", map[string]string{
			"custom": "my-value",
		}},
		{"Bearer custom-one=my-value,custom-two=my-value", map[string]string{
			"custom-one": "my-value",
			"custom-two": "my-value",
		}},
		{"Bearer       white-space=my-value,    foo=bar       ", map[string]string{
			"white-space": "my-value",
			"foo":         "bar",
		}},
	}

	puller := &flatPuller{}

	for _, tt := range imageNameTests {
		t.Run(tt.in, func(t *testing.T) {
			out := puller.parseWwwAuthenticate(tt.in)
			if !reflect.DeepEqual(out, tt.out) {
				t.Fatalf("got %v, want %v", out, tt.out)
			}
		})
	}
}

func TestFlatPuller_FetchToken(t *testing.T) {
	outToken := "foo"

	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.String() {
		case "/v2/auth?service=my-service":
			rw.Write([]byte(fmt.Sprintf("{\"token\": \"%s\"}", outToken)))
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	puller := &flatPuller{
		client: client,
	}

	wwwAuthenticateHeader := fmt.Sprintf("Bearer realm=\"%s/v2/auth\",service=\"my-service\"", server.URL)
	res, err := puller.fetchToken(wwwAuthenticateHeader)
	if err != nil {
		t.Fatalf("puller failed to fetch token: %v", err)
	}

	if res != outToken {
		t.Fatalf("got %q, want %q", res, outToken)
	}
}
