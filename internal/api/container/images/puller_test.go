package images

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestBasicPuller_ParseImageName(t *testing.T) {
	var imageNameTests = []struct {
		in      string
		outHost string
		outRepo string
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

	puller := &basicPuller{}

	for _, tt := range imageNameTests {
		t.Run(tt.in, func(t *testing.T) {
			resultHost, resultRepo, resultRef := puller.parseImageName(tt.in)
			if resultHost != tt.outHost || resultRepo != tt.outRepo || resultRef != tt.outRef {
				t.Fatalf("got %q, %q, %q, want %q, %q, %q", resultHost, resultRepo, resultRef, tt.outHost, tt.outRepo, tt.outRef)
			}
		})
	}
}

func TestBasicPuller_ParseWwwAuthenticate(t *testing.T) {
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

	puller := &basicPuller{}

	for _, tt := range imageNameTests {
		t.Run(tt.in, func(t *testing.T) {
			out := puller.parseWwwAuthenticate(tt.in)
			if !reflect.DeepEqual(out, tt.out) {
				t.Fatalf("got %v, want %v", out, tt.out)
			}
		})
	}
}

func TestBasicPuller_FetchToken(t *testing.T) {
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
	puller := &basicPuller{
		client: client,
	}

	wwwAuthenticateHeader := fmt.Sprintf("Bearer realm=\"%s/v2/auth\",service=\"my-service\"", server.URL)
	res, err := puller.fetchToken(wwwAuthenticateHeader)
	if err != nil {
		t.Fatalf("puller failed to authenticate: %v", err)
	}

	if res != outToken {
		t.Fatalf("got %q, want %q", res, outToken)
	}
}
