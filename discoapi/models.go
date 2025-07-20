package discoapi

type Distribution struct {
	Name         string `json:"name"`
	APIParameter string `json:"api_parameter"`
}

type DistributionsResponse struct {
	Distributions []Distribution `json:"result"`
}
