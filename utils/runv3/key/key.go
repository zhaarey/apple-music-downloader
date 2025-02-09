package wv

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"github.com/gospider007/requests"
	"log/slog"
	"main/utils/runv3/cdm"
)

type Key struct {
	ReqCli        *requests.Client
	BeforeRequest func(cl *requests.Client, preCtx context.Context, method string, href string, options ...requests.RequestOption) (resp *requests.Response, err error)
	AfterRequest  func(*requests.Response) ([]byte, error)
}

func (w *Key) CdmInit() {
	wv.InitConstants()
}
func (w *Key) GetKey(ctx context.Context, licenseServerURL string, PSSH string, headers map[string][]string) (string, []byte, error) {
	initData, err := base64.StdEncoding.DecodeString(PSSH)
	var keybt []byte
	if err != nil {
		slog.Error("pssh decode error: %v", err)
		return "", keybt, err
	}
	cdm, err := wv.NewDefaultCDM(initData)
	if err != nil {
		slog.Error("cdm init error: %v", err)
		return "", keybt, err
	}
	licenseRequest, err := cdm.GetLicenseRequest()
	if err != nil {
		slog.Error("license request error: %v", err)
		return "", keybt, err
	}
	var response *requests.Response
	if w.BeforeRequest != nil {
		response, err = w.BeforeRequest(w.ReqCli, ctx, "post", licenseServerURL, requests.RequestOption{
			Data: licenseRequest,
		})
	} else {
		response, err = w.ReqCli.Request(nil, "post", licenseServerURL, requests.RequestOption{
			Data: licenseRequest,
		})
	}

	if err != nil {
		slog.Error("license request error: %s", err)
		return "", keybt, err
	}
	var licenseResponse []byte
	if w.AfterRequest != nil {
		licenseResponse, err = w.AfterRequest(response)
		if err != nil {
			return "", keybt, err
		}
	} else {
		licenseResponse = response.Content()
	}
	keys, err := cdm.GetLicenseKeys(licenseRequest, licenseResponse)
	command := ""

	for _, key := range keys {
		if key.Type == wv.License_KeyContainer_CONTENT {
			command += hex.EncodeToString(key.ID) + ":" + hex.EncodeToString(key.Value)
			//command += hex.EncodeToString(key.Value)
			keybt = key.Value
		}
	}
	return command, keybt, nil
}
