package deepface

// A simple Golang client for repository: https://github.com/serengil/deepface

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DeepFaceClient struct {
	BaseURL string
}

func NewClient(baseURL string) *DeepFaceClient {
	return &DeepFaceClient{BaseURL: baseURL}
}

func encodeImageToBase64(imgPath string) (string, error) {
	file, err := os.Open(imgPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (c *DeepFaceClient) Represent(modelName, imgPath string) error {
	imgBase64, err := encodeImageToBase64(imgPath)
	if err != nil {
		return err
	}

	data := map[string]string{
		"model_name": modelName,
		"img":        imgBase64,
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(c.BaseURL+"/represent", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response:", string(body))
	return nil
}

func (c *DeepFaceClient) Verify(img1Path, img2Path, modelName, detector, metric string) error {
	img1Base64, err := encodeImageToBase64(img1Path)
	if err != nil {
		return err
	}
	img2Base64, err := encodeImageToBase64(img2Path)
	if err != nil {
		return err
	}

	data := map[string]string{
		"img1":             img1Base64,
		"img2":             img2Base64,
		"model_name":       modelName,
		"detector_backend": detector,
		"distance_metric":  metric,
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(c.BaseURL+"/verify", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response:", string(body))
	return nil
}

func (c *DeepFaceClient) Analyze(imgPath string, actions []string) error {
	imgBase64, err := encodeImageToBase64(imgPath)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"img":     imgBase64,
		"actions": actions,
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(c.BaseURL+"/analyze", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response:", string(body))
	return nil
}
