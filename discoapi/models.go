package discoapi

type Distribution struct {
	Name         string `json:"name"`
	APIParameter string `json:"api_parameter"`
}

type DistributionsResponse struct {
	Distributions []Distribution `json:"result"`
}

type Package struct {
	Id                  string `json:"id"`
	Distribution        string `json:"distribution"`
	JavaVersion         string `json:"java_version"`
	DistributionVersion string `json:"distribution_version"`
}

type PackagesResponse struct {
	Packages []Package `json:"result"`
}

type PackageInfo struct {
	Filename          string `json:"filename"`
	DirectDownloadUri string `json:"direct_download_uri"`
	Checksum          string `json:"checksum"`
	ChecksumType      string `json:"checksum_type"`
}

type PackageInfoResponse struct {
	PackageInfo []PackageInfo `json:"result"`
}
