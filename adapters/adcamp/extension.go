package adcamp

import (
	"fmt"

	"github.com/prebid/prebid-server/openrtb_ext"
)

type extension struct {
	MediaType int8 `json:"mediaType"`
}

func (e *extension) getPrebidMediaType() (t openrtb_ext.BidType, err error) {
	switch e.MediaType {
	case 0:
		t = openrtb_ext.BidTypeBanner
	default:
		err = fmt.Errorf("unknown media type: %d", e.MediaType)
	}

	return
}
