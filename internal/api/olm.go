package api

// CreateOlm creates a new OLM with the specified name
func (c *Client) CreateOlm(name string) (*CreateOlmResponse, error) {
	var response CreateOlmResponse
	request := CreateOlmRequest{
		Name: name,
	}
	err := c.Put("/olm", request, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

