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

	"github.com/opencontainers/go-digest"
	imgspec "github.com/opencontainers/image-spec/specs-go/v1"
)

type RegistryEntity string

const (
	REGISTRY_ENTITY_MANIFEST RegistryEntity = "manifests"
	REGISTRY_ENTITY_BLOB     RegistryEntity = "blobs"
)

type Puller interface {
	PrepareRootFs(id uint32, imageName string) (string, error)
	RemoveRootFs(id uint32) error
}

var _ Puller = &flatPuller{}

type flatPuller struct {
	dir    string
	client *http.Client
	os     string
	arch   string
}

type imageContext struct {
	host  string
	name  string
	token string
}

func NewFlatPuller() (*flatPuller, error) {
	dir := "/var/lib/bx2cloud"

	if err := os.MkdirAll(dir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create the directory that stores rootfs of containers: %w", err)
	}

	return &flatPuller{
		dir:    dir,
		client: http.DefaultClient,
		os:     runtime.GOOS,
		arch:   runtime.GOARCH,
	}, nil
}

func (p *flatPuller) PrepareRootFs(id uint32, imageName string) (string, error) {
	ref, context := p.parseImageName(imageName)

	initialManifestBytes, contentType, err := p.fetchRegistry(ref, REGISTRY_ENTITY_MANIFEST, context)
	if err != nil {
		return "", fmt.Errorf("failed to fetch initial manifest: %w", err)
	}

	var manifest imgspec.Manifest
	switch contentType {
	case "application/vnd.docker.distribution.manifest.list.v2+json":
		fallthrough
	case imgspec.MediaTypeImageIndex: // We discovered a manifest index, need to locate the correct image manifest
		var index imgspec.Index
		if err := json.Unmarshal(initialManifestBytes, &index); err != nil {
			return "", fmt.Errorf("failed to decode index: %w", err)
		}

		digest, err := p.findManifestDigestInIndex(&index)
		if err != nil {
			return "", err
		}

		manifestBytes, _, err := p.fetchRegistry(digest.String(), REGISTRY_ENTITY_MANIFEST, context)
		if err != nil {
			return "", fmt.Errorf("failed to fetch manifest: %w", err)
		}

		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return "", fmt.Errorf("failed to decode manifest: %w", err)
		}
	case "application/vnd.docker.distribution.manifest.v2+json":
		fallthrough
	case imgspec.MediaTypeImageManifest:
		if err := json.Unmarshal(initialManifestBytes, &manifest); err != nil {
			return "", fmt.Errorf("failed to decode manifest: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported initial manifest content type %q", contentType)
	}

	if manifest.Config.MediaType != imgspec.MediaTypeImageConfig &&
		manifest.Config.MediaType != "application/vnd.docker.container.image.v1+json" {
		return "", fmt.Errorf("unsupported config content type %q", contentType)
	}

	configBytes, _, err := p.fetchRegistry(manifest.Config.Digest.String(), REGISTRY_ENTITY_BLOB, context)
	if err != nil {
		return "", fmt.Errorf("failed to fetch config: %w", err)
	}

	var config imgspec.Image
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return "", fmt.Errorf("failed to decode config: %w", err)
	}

	if config.Platform.OS != p.os || config.Platform.Architecture != p.arch {
		return "", fmt.Errorf("failed to find an image for %s/%s", p.os, p.arch)
	}

	// TODO: (This PR) take config.Config and turn into libcontainer.Process

	rootfsDir := p.getRootFsDir(id)
	if _, err := os.Stat(rootfsDir); err == nil {
		return "", fmt.Errorf("something already exsits at the rootfs path %q", rootfsDir)
	}

	for _, layer := range manifest.Layers {
		err := p.fetchAndUnpackLayer(layer.Digest.String(), context, rootfsDir)
		if err != nil {
			return "", fmt.Errorf("failed to fetch and unpack layer: %w", err)
		}
	}

	{
		tmpPath := filepath.Join(rootfsDir, "tmp")
		_ = os.Chmod(tmpPath, 0o777|os.ModeSticky)
	}

	return rootfsDir, nil
}

func (p *flatPuller) getRootFsDir(id uint32) string {
	return filepath.Join(p.dir, strconv.FormatUint(uint64(id), 10))
}

func (p *flatPuller) requestRegistry(ref string, entity RegistryEntity, context *imageContext) (*http.Response, error) {
	url := fmt.Sprintf("https://%s/v2/%s/%s/%s", context.host, context.name, entity, ref)
	createRequest := func() (*http.Request, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		if context.token != "" {
			req.Header.Set("Authorization", "Bearer "+context.token)
		}
		return req, nil
	}

	req, err := createRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to create the request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	// Maybe we need to authenticate
	if context.token == "" && resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		wwwAuthenticateHeader := resp.Header.Get("www-authenticate")
		if wwwAuthenticateHeader == "" {
			return nil, fmt.Errorf("registry requested to authenticate, but did not include www-authenticate header")
		}

		context.token, err = p.fetchToken(wwwAuthenticateHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate: %w", err)
		}

		req, err := createRequest()
		if err != nil {
			return nil, fmt.Errorf("failed to create the request: %w", err)
		}

		resp, err = p.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
		}
	}

	if resp.StatusCode != http.StatusOK {
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fetch failed with status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("fetch failed with status %d: %s", resp.StatusCode, string(bytes))
	}

	return resp, nil
}

func (p *flatPuller) fetchRegistry(
	ref string,
	entity RegistryEntity,
	context *imageContext,
) ([]byte, string, error) {
	resp, err := p.requestRegistry(ref, entity, context)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read body: %w", err)
	}

	return bytes, resp.Header.Get("Content-Type"), nil
}

func (p *flatPuller) fetchAndUnpackLayer(ref string, context *imageContext, dir string) error {
	resp, err := p.requestRegistry(ref, REGISTRY_ENTITY_BLOB, context)
	if err != nil {
		return fmt.Errorf("failed to download layer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch failed with status %d", resp.StatusCode)
	}

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
				return err
			}
		case tar.TypeReg:
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			// Remove a file (most likely a symlink) before symlinking if it existed before
			if _, err := os.Lstat(targetPath); err == nil {
				if err := os.Remove(targetPath); err != nil {
					return err
				}
			}

			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		case tar.TypeLink:
			oldname := filepath.Join(dir, header.Linkname)

			if err := os.Link(oldname, targetPath); err != nil {
				return err
			}
		default:
			log.Printf("Skipping unsupported tar entry type %c for file %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func (p *flatPuller) fetchToken(wwwAuthenticateHeader string) (string, error) {
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

func (p *flatPuller) parseImageName(imageName string) (ref string, context *imageContext) {
	context = &imageContext{}
	parts := strings.SplitN(imageName, "/", 2)
	const defaultHost = "registry-1.docker.io"

	var repoAndRef string
	if len(parts) != 2 {
		context.host = defaultHost
		repoAndRef = imageName
	} else {
		// Determine if the '/' was a part of the repository or separated the host part
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			context.host, repoAndRef = parts[0], parts[1]
		} else {
			context.host = defaultHost
			repoAndRef = imageName
		}
	}

	repoParts := strings.SplitN(repoAndRef, ":", 2)
	if len(repoParts) != 2 {
		context.name = repoAndRef
		ref = "latest"
	} else {
		context.name, ref = repoParts[0], repoParts[1]
	}

	if strings.Contains(context.host, "docker.io") && !strings.Contains(context.name, "/") {
		context.name = "library/" + context.name
	}

	return
}

func (p *flatPuller) parseWwwAuthenticate(header string) map[string]string {
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

func (p *flatPuller) findManifestDigestInIndex(index *imgspec.Index) (*digest.Digest, error) {
	for _, manifest := range index.Manifests {
		if manifest.Platform.OS != p.os || manifest.Platform.Architecture != p.arch {
			continue
		}

		return &manifest.Digest, nil
	}

	return nil, fmt.Errorf("failed to find an image for %s/%s in the index", p.os, p.arch)
}

func (p *flatPuller) RemoveRootFs(id uint32) error {
	return os.RemoveAll(p.getRootFsDir(id))
}
