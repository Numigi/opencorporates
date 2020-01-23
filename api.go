// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// Package opencorporates is an unofficial Golang API Client for the OpenCorporates.
// http://api.opencorporates.com/documentation/API-Reference
package opencorporates

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Date represents a date without time.
type Date struct {
	Time time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (m *Date) UnmarshalJSON(b []byte) (err error) {
	data := strings.Trim(string(b), "\"")
	if data == "null" {
		return nil
	}
	m.Time, err = time.Parse("2006-01-02", data)
	return err
}

type request struct {
	nb int
	sync.RWMutex
}

func (r *request) incr() {
	r.Lock()
	r.nb++
	r.Unlock()
}

// Client represents the API Client.
type Client struct {
	Version,
	Token string
	http Getter
	rq   *request
}

// API returns a new instance of the Client with http default Client.
func API() *Client {
	return &Client{
		http: http.DefaultClient,
		rq:   &request{},
	}
}

// Request returns the number of call done.
func (api *Client) RequestCount() int {
	api.rq.RLock()
	defer api.rq.RUnlock()
	return api.rq.nb
}

// Companies returns an iterator of companies with its name or / and in this jurisdiction.
func (api *Client) Companies(name, jurisdiction string) *CompanyIterator {
	return &CompanyIterator{
		Api:          api,
		Page:         NewPager(1),
		Name:         name,
		Jurisdiction: jurisdiction,
	}
}

// CompanyByID returns the company by its identifier and jurisdiction code.
// companies/fr/529591737
func (api *Client) CompanyByID(id, jurisdiction string) (c Company, err error) {
	url, err := api.url(ByNumberURL, id, jurisdiction)
	if err != nil {
		return
	}
	resp, err := api.call(url)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Response contains the search response.
	type Response struct {
		Results struct {
			Company Company `json:"company"`
		} `json:"results"`
	}
	var res Response
	if err = json.NewDecoder(resp.Body).Decode(&res); err == nil {
		c = res.Results.Company
	}
	return
}

func (api *Client) call(url string) (resp *http.Response, err error) {

	// And increments the counter of request.
	api.rq.incr()
	resp, err = api.http.Get(url)
	if err != nil {
		return
	}
	// Only accepts valid response.
	if resp.StatusCode > http.StatusBadRequest {
		if resp.StatusCode < http.StatusInternalServerError {
			// Response contains the search response.
			type Response struct {
				Err struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			var res Response
			if jErr := json.NewDecoder(resp.Body).Decode(&res); jErr == nil {
				err = errors.New(res.Err.Message)
			}
		}
		if err == nil {
			err = errors.New(resp.Status)
		}
		_ = resp.Body.Close()
	}
	return
}

// Getter represents the mean to do a HTTP get.
type Getter interface {
	Get(url string) (*http.Response, error)
}

// UseClient allows to use your own HTTP Client to request the API.
func (api *Client) UseClient(http Getter) *Client {
	api.http = http
	return api
}
