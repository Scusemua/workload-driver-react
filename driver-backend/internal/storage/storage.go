package storage

import "github.com/goccy/go-json"

type RemoteStorage struct {
	// Name is the name of the RemoteStorage.
	Name string `json:"name" mapstructure:"name"`

	// DownloadRate is the average rate, in bytes per second, that data is downloaded from the RemoteStorage.
	DownloadRate int64 `json:"download_rate" mapstructure:"download_rate"`

	// UploadRate is the average rate, in bytes per second, that data is uploaded to the RemoteStorage.
	UploadRate int64 `json:"upload_rate" mapstructure:"upload_rate"`

	// DownloadVariancePercent is the maximum amount by which the download rate can vary/deviate
	// from its set value during a simulated I/O operation.
	DownloadVariancePercent float64 `json:"download_variance_percent" mapstructure:"download_variance_percent"`

	// UploadVariancePercent is the maximum amount by which the upload rate can vary/deviate from
	// its set value during a simulated I/O operation.
	UploadVariancePercent float64 `json:"upload_variance_percent" mapstructure:"upload_variance_percent"`

	// ReadFailureChancePercentage is the likelihood as a percentage (value between 0 and 1) that an
	// error occurs during any single read operation.
	ReadFailureChancePercentage float64 `json:"read_failure_chance_percentage" mapstructure:"read_failure_chance_percentage"`

	// WriteFailureChancePercentage is the likelihood as a percentage (value between 0 and 1) that an
	// error occurs during any single write operation.
	WriteFailureChancePercentage float64 `json:"write_failure_chance_percentage" mapstructure:"write_failure_chance_percentage"`
}

func (rs *RemoteStorage) String() string {
	m, err := json.Marshal(rs)
	if err != nil {
		panic(err)
	}

	return string(m)
}

func (rs *RemoteStorage) StringFormatted() string {
	m, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(m)
}
