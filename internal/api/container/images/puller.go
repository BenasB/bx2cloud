package images

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Puller interface {
	PrepareRootFs(id uint32, imageName string) (string, error)
}

var _ Puller = &basicPuller{}

type basicPuller struct {
	dir    string
	client *http.Client
}

func NewBasicPuller() (*basicPuller, error) {
	dir := "/var/lib/bx2cloud"

	if err := os.MkdirAll(dir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create the directory that stores rootfs of containers: %w", err)
	}

	return &basicPuller{
		dir:    dir,
		client: http.DefaultClient,
	}, nil
}

func (p *basicPuller) PrepareRootFs(id uint32, imageName string) (string, error) {
	host, repo, ref := p.parseImageName(imageName)

	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, ref)

	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create the manifest request: %w", err)
	}
	const acceptHeader = "application/vnd.oci.image.manifest.v1+json"
	req.Header.Set("Accept", acceptHeader)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuthenticateHeader := resp.Header.Get("www-authenticate")
		if wwwAuthenticateHeader == "" {
			return "", fmt.Errorf("registry requested to authenticate, but did not include www-authenticate header")
		}

		token, err := p.requestToken(wwwAuthenticateHeader)
		if err != nil {
			return "", fmt.Errorf("failed to authenticate: %w", err)
		}

		req, err := http.NewRequest("GET", manifestURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create the authenticated manifest request: %w", err)
		}
		req.Header.Set("Accept", acceptHeader)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = p.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to fetch authenticated manifest: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("manifest fetch failed with status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest body: %w", err)
	}

	log.Print(string(bodyBytes))

	return "", nil // TODO: return real path to rootfs
}

func (p *basicPuller) requestToken(wwwAuthenticateHeader string) (string, error) {
	params := p.parseWwwAuthenticate(wwwAuthenticateHeader)
	realm, ok := params["realm"]
	if !ok {
		return "", fmt.Errorf("www-authenticate header missing 'realm'")
	}

	tokenRequestUrl, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf("'realm' could not be parsed into a URL")
	}

	query := tokenRequestUrl.Query()
	for k, v := range params {
		if k == "realm" {
			continue
		}

		query.Set(k, v)
	}
	tokenRequestUrl.RawQuery = query.Encode()

	resp, err := p.client.Get(tokenRequestUrl.String())
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response JSON: %w", err)
	}

	token, ok := result["token"]
	if !ok {
		return "", fmt.Errorf("token not found in the authentication JSON response: %w", err)
	}

	return fmt.Sprintf("%v", token), nil
}

func (p *basicPuller) parseImageName(imageName string) (host, repo, ref string) {
	parts := strings.SplitN(imageName, "/", 2)
	const defaultHost = "registry-1.docker.io"

	var repoAndRef string
	if len(parts) != 2 {
		host = defaultHost
		repoAndRef = imageName
	} else {
		// Determine if the '/' was a part of the repository or separated the host part
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			host, repoAndRef = parts[0], parts[1]
		} else {
			host = defaultHost
			repoAndRef = imageName
		}
	}

	repoParts := strings.SplitN(repoAndRef, ":", 2)
	if len(repoParts) != 2 {
		repo = repoAndRef
		ref = "latest"
	} else {
		repo, ref = repoParts[0], repoParts[1]
	}

	if strings.Contains(host, "docker.io") && !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}

	return
}

func (p *basicPuller) parseWwwAuthenticate(header string) map[string]string {
	parts := strings.Split(strings.TrimPrefix(header, "Bearer "), ",")
	params := make(map[string]string)
	for _, part := range parts {
		keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(keyValue) == 2 {
			params[keyValue[0]] = strings.Trim(keyValue[1], "\"")
		}
	}
	return params
}
