package oci

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	spec "github.com/opencontainers/image-spec/specs-go/v1"
)

type PushBlobOptions struct {
	Digest   spec.Descriptor
	File     []byte
	Name     string
	Insecure bool
	Tag      Tag
}

type PullBlobOptions struct {
	Digest spec.Descriptor
	Name   string
	Tag    *Tag
}

type PushManifestOptions struct {
	Manifest *spec.Manifest
	Tag      *Tag
	Insecure bool
}

type OciClient struct {
	Credentials *OciCredentials
}

type OciCredentials struct {
	Username string
	Password string
	encoded  string
}

func (c *OciClient) PushBlob(opts PushBlobOptions) error {
	var protocol string
	if opts.Insecure {
		protocol = "http"
	} else {
		protocol = "https"
	}

	var endpoint string
	if opts.Tag.Namespace != "" {
		endpoint = fmt.Sprintf("%s://%s/v2/%s/%s/blobs/uploads/", protocol, &opts.Tag.Host, opts.Tag.Namespace, opts.Tag.Name)
	} else {
		endpoint = fmt.Sprintf("%s://%s/v2/%s/blobs/uploads/", protocol, &opts.Tag.Host, opts.Tag.Name)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %s", err.Error())
	}

	if c.Credentials != nil {
		req.Header.Add("Authorization", c.Credentials.encoded)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err.Error())
	}

	if resp.StatusCode != 202 {
		return fmt.Errorf("failed to push blob: %s", resp.Status)
	}

	location := resp.Header.Get("Location")
	req, err = http.NewRequest("PUT", location, bytes.NewReader(opts.File))
	if err != nil {
		return fmt.Errorf("error uploading blob: %s", err.Error())
	}

	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Content-Length", fmt.Sprintf("%d", len(opts.File)))
	query := req.URL.Query()
	query.Add("digest", opts.Digest.Digest.String())
	req.URL.RawQuery = query.Encode()

	if c.Credentials != nil {
		req.Header.Add("Authorization", c.Credentials.encoded)
	}

	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("unauthorized, please use nori login to authenticate")
		}
		return fmt.Errorf("failed to push blob: %s", resp.Status)
	}
	return nil
}

func (c *OciClient) PullBlob(opts PullBlobOptions) ([]byte, error) {
	var endpoint string
	if opts.Tag.Namespace != "" {
		endpoint = fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", opts.Tag.Host, opts.Tag.Namespace, opts.Tag.Name, opts.Digest.Digest)
	} else {
		endpoint = fmt.Sprintf("https://%s/v2/%s/blobs/%s", opts.Tag.Host, opts.Tag.Name, opts.Digest.Digest)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err.Error())
	}

	if c.Credentials != nil {
		req.Header.Add("Authorization", c.Credentials.encoded)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err.Error())
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unauthorized, please use nori login to authenticate")
		}
		return nil, fmt.Errorf("failed to pull blob: %s", resp.Status)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading blob: %s", err.Error())
	}

	return data, nil
}

func (c *OciClient) PullManifest(tag *Tag) (*spec.Manifest, error) {
	var api_endpoint string
	if tag.Host == "" {
		return nil, fmt.Errorf("Host is required, but not provided")
	}

	if tag.Namespace != "" {
		api_endpoint = fmt.Sprintf("https://%s/%s/%s/v2/%s/%s/manifests/%s", tag.Host, tag.Namespace, tag.Name, tag.Version, tag.Name, tag.Version)
	} else {
		api_endpoint = fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", tag.Host, tag.Name, tag.Version, tag.Name, tag.Version)
	}

	req, err := http.NewRequest("GET", api_endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", spec.MediaTypeImageManifest)
	if c.Credentials != nil {
		req.Header.Add("Authorization", c.Credentials.encoded)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unauthorized, please use nori login to authenticate")
		}

		return nil, fmt.Errorf("cannot to pull manifest: %s", resp.Status)
	}

	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	manifest := &spec.Manifest{}
	err = json.Unmarshal(manifestBytes, manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func (c *OciClient) PushManifest(opts PushManifestOptions) error {
	var protocol string
	var endpoint string
	jsonBytes, err := json.Marshal(opts.Manifest)
	if err != nil {
		return err
	}

	if opts.Insecure {
		protocol = "http"
	} else {
		protocol = "https"
	}

	if opts.Tag.Namespace != "" {
		endpoint = fmt.Sprintf("%s://%s/v2/%s/%s/manifests/%s", protocol, opts.Tag.Host, opts.Tag.Namespace, opts.Tag.Name, opts.Tag.Version)
	} else {
		endpoint = fmt.Sprintf("%s://%s/v2/%s/manifests/%s", protocol, opts.Tag.Host, opts.Tag.Name, opts.Tag.Version)
	}

	req, err := http.NewRequest("HEAD", endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %s", err.Error())
	}

	if c.Credentials != nil {
		req.Header.Add("Authorization", c.Credentials.encoded)
	}

	req.Header.Add("Content-Type", spec.MediaTypeImageManifest)
	req.Header.Add("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		uploadReq, err := http.NewRequest("PUT", endpoint, bytes.NewReader(jsonBytes))
		if err != nil {
			return fmt.Errorf("error creating request: %s", err.Error())
		}

		uploadReq.Header.Add("Content-Type", spec.MediaTypeImageManifest)
		uploadReq.Header.Add("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))

		if c.Credentials != nil {
			uploadReq.Header.Add("Authorization", c.Credentials.encoded)
		}

		resp, err = client.Do(uploadReq)
		if err != nil {
			return fmt.Errorf("error sending request: %s", err.Error())
		}

		if resp.StatusCode != 201 {
			if resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("unauthorized, please use nori login to authenticate")
			}
			return fmt.Errorf("failed to push manifest: %s", resp.Status)
		}
	}

	return nil
}

func (c *OciClient) SetBasicAuth(username, password string) {
	userpass := fmt.Sprintf("%s:%s", username, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(userpass))
	authHeader := fmt.Sprintf("Basic %s", encoded)
	c.Credentials = &OciCredentials{
		Username: username,
		Password: password,
		encoded:  authHeader,
	}
}

func (c *OciClient) GetCredentials() *OciCredentials {
	return c.Credentials
}

func (c *OciClient) SetCredentials(creds OciCredentials) {
	c.Credentials = &creds
}

func NewOciClient() *OciClient {
	return &OciClient{}
}
