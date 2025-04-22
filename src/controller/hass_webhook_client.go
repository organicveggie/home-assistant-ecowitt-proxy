/*
Copyright © 2023-2024 Sean Laurent <o r g a n i c v e g g i e @ Google Mail>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type HassWebhookClient struct {
	authToken string
	formData  url.Values
	url       string

	openClientFn HassOpenHttpFn
}

type HassClientOption func(*HassWebhookClient)

func WithOpenClientFn(openHttpClientFn HassOpenHttpFn) HassClientOption {
	return func(hc *HassWebhookClient) {
		hc.openClientFn = openHttpClientFn
	}
}

type HassOpenHttpFn func() *http.Client

func defaultHassOpenHttpFn() *http.Client {
	return &http.Client{}
}

func NewHassClient(url string, authToken string, formData url.Values, opts ...HassClientOption) *HassWebhookClient {
	hc := &HassWebhookClient{
		authToken:    authToken,
		formData:     formData,
		url:          url,
		openClientFn: defaultHassOpenHttpFn,
	}

	for _, opt := range opts {
		opt(hc)
	}
	return hc
}

func (hc *HassWebhookClient) PostData(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", hc.url, strings.NewReader(hc.formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating HTTP request for %s: %w", hc.url, err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hc.authToken))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := hc.openClientFn()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to %q: %w", hc.url, err)
	}

	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respMsg := buf.String()

		return fmt.Errorf("error making request to %q. Response code: %d. Response: %s",
			hc.url, resp.StatusCode, respMsg)
	}

	return nil
}
