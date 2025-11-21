package wv

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"log/slog"

	"github.com/go-resty/resty/v2"

	wv "main/utils/runv3/cdm"
)

type Key struct {
	ReqCli *resty.Client

	BeforeRequest func(cl *resty.Client, ctx context.Context, url string, body []byte) (*resty.Response, error)

	AfterRequest func(*resty.Response) ([]byte, error)
}

func (w *Key) CdmInit() {
	wv.InitConstants()
}

func (w *Key) GetKey(ctx context.Context, licenseServerURL string, PSSH string, headers map[string][]string) (string, []byte, error) {
	initData, err := base64.StdEncoding.DecodeString(PSSH)
	var keybt []byte
	if err != nil {
		slog.Error("pssh decode error", slog.Any("err", err))
		return "", keybt, err
	}
	cdm, err := wv.NewDefaultCDM(initData)
	if err != nil {
		slog.Error("cdm init error", slog.Any("err", err))
		return "", keybt, err
	}
	licenseRequest, err := cdm.GetLicenseRequest()
	if err != nil {
		slog.Error("license request error", slog.Any("err", err))
		return "", keybt, err
	}

	var response *resty.Response

	if w.BeforeRequest != nil {
		response, err = w.BeforeRequest(w.ReqCli, ctx, licenseServerURL, licenseRequest)
	} else {
		response, err = w.ReqCli.R().
			SetContext(ctx).
			SetBody(licenseRequest).
			Post(licenseServerURL)
	}

	if err != nil {
		slog.Error("license request error", slog.Any("err", err))
		return "", keybt, err
	}

	var licenseResponse []byte
	if w.AfterRequest != nil {
		licenseResponse, err = w.AfterRequest(response)
		if err != nil {
			return "", keybt, err
		}
	} else {
		licenseResponse = response.Body()
	}

	keys, err := cdm.GetLicenseKeys(licenseRequest, licenseResponse)
	command := ""

	for _, key := range keys {
		if key.Type == wv.License_KeyContainer_CONTENT {
			command += hex.EncodeToString(key.Value)
			keybt = key.Value
		}
	}
	return command, keybt, nil
}
