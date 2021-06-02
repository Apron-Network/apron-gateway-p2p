package models

import "fmt"

func (req *ApronServiceRequest) ServiceUrlWithSchema() string {
	return fmt.Sprintf("%s://%s", req.Schema, req.ServiceUrl)
}