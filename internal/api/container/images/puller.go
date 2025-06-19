package images

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Puller interface {
	PrepareRootFs(id uint32, imageName string) (string, error)
}

var _ Puller = &basicPuller{}

type basicPuller struct {
	dir    string
	client *http.Client
	os     string
	arch   string
}

func NewBasicPuller() (*basicPuller, error) {
	dir := "/var/lib/bx2cloud"

	if err := os.MkdirAll(dir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create the directory that stores rootfs of containers: %w", err)
	}

	return &basicPuller{
		dir:    dir,
		client: http.DefaultClient,
		os:     runtime.GOOS,
		arch:   runtime.GOARCH,
	}, nil
}

// TODO: (This PR) Use github.com/opencontainers/image-spec
type OciIndex struct {
	Manifests []struct {
		Digest   string `json:"digest"`
		Platform struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

type OciManifest struct {
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		Digest string `json:"digest"`
	} `json:"layers"`
}

func (p *basicPuller) PrepareRootFs(id uint32, imageName string) (string, error) {
	host, repo, ref := p.parseImageName(imageName)

	// TODO: (This PR) Handle images which do not have an index
	indexURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, ref)
	index, token, err := p.fetchIndex(indexURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch index: %w", err)
	}
	log.Print(index)

	var manifestDigest string
	for _, manifest := range index.Manifests {
		if manifest.Platform.OS != p.os || manifest.Platform.Architecture != p.arch {
			continue
		}

		manifestDigest = manifest.Digest
		break
	}

	if manifestDigest == "" {
		return "", fmt.Errorf("failed to find an image for %s/%s in the index", p.os, p.arch)
	}

	manifestUrl := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, manifestDigest)
	manifest, err := p.fetchManifest(manifestUrl, token)
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest: %w", err)
	}
	log.Print(manifest)

	// TODO: (This PR) Pull config as well

	rootfsDir := filepath.Join(p.dir, strconv.FormatUint(uint64(id), 10))
	for _, layer := range manifest.Layers {
		layerUrl := fmt.Sprintf("https://%s/v2/%s/blobs/%s", host, repo, layer.Digest)
		err := p.fetchAndUnpackLayer(layerUrl, token, rootfsDir)
		if err != nil {
			return "", fmt.Errorf("failed to fetch and unpack layer: %w", err)
		}
	}

	return rootfsDir, nil
}

func (p *basicPuller) fetchIndex(url string) (*OciIndex, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create the index request: %w", err)
	}
	const acceptHeader = "application/vnd.oci.image.manifest.v1+json"
	req.Header.Set("Accept", acceptHeader)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to perform index fetching HTTP request: %w", err)
	}
	defer resp.Body.Close()

	var token string
	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuthenticateHeader := resp.Header.Get("www-authenticate")
		if wwwAuthenticateHeader == "" {
			return nil, "", fmt.Errorf("registry requested to authenticate, but did not include www-authenticate header")
		}

		token, err = p.fetchToken(wwwAuthenticateHeader)
		if err != nil {
			return nil, "", fmt.Errorf("failed to authenticate: %w", err)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create the authenticated index request: %w", err)
		}
		req.Header.Set("Accept", acceptHeader)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = p.client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("failed to fetch authenticated index: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("index fetch failed with status %d", resp.StatusCode)
	}

	indexBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read index body: %w", err)
	}

	var index OciIndex
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		return nil, "", fmt.Errorf("failed to decode index: %w", err)
	}

	return &index, token, nil
}

func (p *basicPuller) fetchManifest(url string, token string) (*OciManifest, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create the manifest request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform manifest fetching HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest fetch failed with status %d", resp.StatusCode)
	}

	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest body: %w", err)
	}

	var manifest OciManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	return &manifest, nil
}

func (p *basicPuller) fetchAndUnpackLayer(url string, token string, dir string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create the blob request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download layer: %w", err)
	}
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dir, header.Name)
		baseName := filepath.Base(header.Name)

		const whiteoutPrefix = ".wh."
		if strings.HasPrefix(baseName, whiteoutPrefix) {
			originalFileName := strings.TrimPrefix(baseName, whiteoutPrefix)
			pathToRemove := filepath.Join(filepath.Dir(targetPath), originalFileName)
			if err := os.RemoveAll(pathToRemove); err != nil {
				return fmt.Errorf("could not process whiteout for %s: %w", pathToRemove, err)
			}
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				log.Print(targetPath)
				return err
			}
		case tar.TypeReg:
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				log.Print(targetPath)
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				log.Print(targetPath)
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				log.Print(targetPath)
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			if _, err := os.Lstat(targetPath); err == nil {
				if err := os.Remove(targetPath); err != nil {
					return err
				}
			}

			// Ensure symlinked path exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				log.Print(targetPath)
				return err
			}

			var oldname string
			if filepath.IsAbs(header.Linkname) {
				oldname = filepath.Join(dir, header.Linkname)
			} else {
				oldname = filepath.Join(filepath.Dir(targetPath), header.Linkname)
			}

			if err := os.Symlink(oldname, targetPath); err != nil {
				log.Print(dir)
				log.Print(header.Linkname)
				log.Print(oldname)
				log.Print(targetPath)
				return err
			}
		case tar.TypeLink:
			var oldname string
			if filepath.IsAbs(header.Linkname) {
				oldname = filepath.Join(dir, header.Linkname)
			} else {
				oldname = filepath.Join(filepath.Dir(targetPath), header.Linkname)
			}

			if err := os.Link(oldname, targetPath); err != nil {
				log.Print(targetPath)
				return err
			}
		default:
			log.Printf("Skipping unsupported tar entry type %c for file %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func (p *basicPuller) fetchToken(wwwAuthenticateHeader string) (string, error) {
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
