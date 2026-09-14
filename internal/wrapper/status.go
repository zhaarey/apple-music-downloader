package wrapper

// Status queries the wrapper-lite /status endpoint and returns the supported regions.
func (c *Client) Status() ([]string, error) {
	endpoint := "/status"
	body, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}
	data, err := decodeEnvelope[StatusData](body, endpoint)
	if err != nil {
		return nil, err
	}
	return data.Regions, nil
}

// GetStatus queries wrapper-lite's /status endpoint using baseURL and returns the supported regions.
func GetStatus(baseURL string) ([]string, error) {
	return New(baseURL).Status()
}
